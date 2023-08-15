// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerlib

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/imagecustomizerapi"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/imagegen/configuration"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/imagegen/diskutils"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/safechroot"
	"github.com/stretchr/testify/assert"
)

func TestCustomizeImageEmptyConfig(t *testing.T) {
	var err error

	buildDir := filepath.Join(tmpDir, "TestCustomizeImageEmptyConfig")
	outImageFilePath := filepath.Join(buildDir, "image.vhd")

	// Create empty disk.
	diskFilePath, _, _, err := createFakeEfiImage(buildDir)
	if !assert.NoError(t, err) {
		return
	}

	// Customize image.
	err = CustomizeImage(buildDir, buildDir, &imagecustomizerapi.SystemConfig{}, diskFilePath, outImageFilePath, "vhd")
	if !assert.NoError(t, err) {
		return
	}

	// Check output file type.
	checkFileType(t, outImageFilePath, "vhd")
}

func TestCustomizeImageCopyFiles(t *testing.T) {
	var err error

	buildDir := filepath.Join(tmpDir, "TestCustomizeImageCopyFiles")
	configFile := filepath.Join(testDir, "addfiles-config.yaml")
	outImageFilePath := filepath.Join(buildDir, "image.qcow2")

	// Create empty disk.
	diskFilePath, newMountDirectories, mountPoints, err := createFakeEfiImage(buildDir)
	if !assert.NoError(t, err) {
		return
	}

	// Customize image.
	err = CustomizeImageWithConfigFile(buildDir, configFile, diskFilePath, outImageFilePath, "raw")
	if !assert.NoError(t, err) {
		return
	}

	// Check output file type.
	checkFileType(t, outImageFilePath, "raw")

	// Mount the output disk image so that its contents can be checked.
	diskDevPath, err := diskutils.SetupLoopbackDevice(outImageFilePath)
	if !assert.NoError(t, err) {
		return
	}
	defer diskutils.DetachLoopbackDevice(diskDevPath)

	imageChroot := safechroot.NewChroot(filepath.Join(buildDir, "imageroot"), false)
	err = imageChroot.Initialize("", newMountDirectories, mountPoints)
	if !assert.NoError(t, err) {
		return
	}
	defer imageChroot.Close(false)

	// Check the contents of the copied file.
	file_contents, err := os.ReadFile(filepath.Join(imageChroot.RootDir(), "a.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "abcdefg\n", string(file_contents))
}

func createFakeEfiImage(buildDir string) (string, []string, []*safechroot.MountPoint, error) {
	var err error

	err = os.MkdirAll(buildDir, os.ModePerm)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to make build directory (%s): %w", buildDir, err)
	}

	// Use a prototypical Mariner image partition config.
	diskConfig := configuration.Disk{
		PartitionTableType: configuration.PartitionTableTypeGpt,
		MaxSize:            4096,
		Partitions: []configuration.Partition{
			{
				ID:     "boot",
				Flags:  []configuration.PartitionFlag{"esp", "boot"},
				Start:  1,
				End:    9,
				FsType: "fat32",
			},
			{
				ID:     "rootfs",
				Start:  9,
				End:    0,
				FsType: "ext4",
			},
		},
	}

	// Create raw disk image file.
	rawDisk, err := diskutils.CreateEmptyDisk(buildDir, "disk.raw", diskConfig)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create empty disk file in (%s): %w", buildDir, err)
	}

	// Connect raw disk image file.
	diskDevPath, err := diskutils.SetupLoopbackDevice(rawDisk)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to mount raw disk (%s) as a loopback device: %w", rawDisk, err)
	}
	defer diskutils.DetachLoopbackDevice(diskDevPath)

	// Set up partitions.
	_, _, _, _, err = diskutils.CreatePartitions(diskDevPath, diskConfig,
		configuration.RootEncryption{}, configuration.ReadOnlyVerityRoot{})
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create partitions on disk (%s): %w", diskDevPath, err)
	}

	// Create partition mount config.
	newMountDirectories := []string{}
	mountPoints := []*safechroot.MountPoint{
		safechroot.NewPreDefaultsMountPoint(fmt.Sprintf("%sp2", diskDevPath), "/", "ext4", 0, ""),
		safechroot.NewMountPoint(fmt.Sprintf("%sp1", diskDevPath), "/boot", "vfat", 0, ""),
	}

	// Mount the partitions.
	imageChroot := safechroot.NewChroot(filepath.Join(buildDir, "imageroot"), false)
	err = imageChroot.Initialize("", newMountDirectories, mountPoints)
	if err != nil {
		return "", nil, nil, err
	}
	defer imageChroot.Close(false)

	// Get the UUID of the OS partition.
	diskPartitions, err := diskutils.GetDiskPartitions(diskDevPath)
	if err != nil {
		return "", nil, nil, err
	}

	var osPartition *diskutils.PartitionInfo
	for _, diskPartition := range diskPartitions {
		if diskPartition.Mountpoint == imageChroot.RootDir() {
			osPartition = &diskPartition
			break
		}
	}

	if osPartition == nil {
		return "", nil, nil, fmt.Errorf("os partition not found (%s)", diskDevPath)
	}

	// Write a fake grub.cfg file so that the partition discovery logic works.
	grubConfigDirectory := filepath.Join(imageChroot.RootDir(), "/boot/boot/grub2")

	err = os.MkdirAll(grubConfigDirectory, os.ModePerm)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create grub.cfg directory: %w", err)
	}

	grubConfig := fmt.Sprintf("search -n -u %s -s\n", osPartition.Uuid)

	err = os.WriteFile(filepath.Join(grubConfigDirectory, "grub.cfg"), []byte(grubConfig), os.ModePerm)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to write fake grub.cfg file: %w", err)
	}

	return rawDisk, newMountDirectories, mountPoints, nil
}

func checkFileType(t *testing.T, filePath string, expectedFileType string) {
	fileType, err := getImageFileType(filePath)
	assert.NoError(t, err)
	assert.Equal(t, expectedFileType, fileType)
}

func getImageFileType(filePath string) (string, error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0)
	if err != nil {
		return "", err
	}
	defer file.Close()

	firstBytes := make([]byte, 512)
	readByteCount, err := file.Read(firstBytes)
	if err != nil {
		return "", err
	}

	switch {
	case readByteCount >= 8 && bytes.Equal(firstBytes[:8], []byte("conectix")):
		return "vhd", nil

	case readByteCount >= 8 && bytes.Equal(firstBytes[:8], []byte("vhdxfile")):
		return "vhdx", nil

	// Check for the MBR signature (which exists even on GPT formatted drives).
	case readByteCount >= 512 && bytes.Equal(firstBytes[510:512], []byte{0x55, 0xAA}):
		return "raw", nil
	}

	return "", fmt.Errorf("unknown file type: %s", filePath)
}

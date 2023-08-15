// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerlib

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/imagecustomizerapi"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/imagegen/diskutils"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/safechroot"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/shell"
	"golang.org/x/sys/unix"
)

var (
	rootfsPartitionRegex = regexp.MustCompile(`(?m)^search -n -u ([a-zA-Z0-9\-]+) -s$`)
)

func CustomizeImageWithConfigFile(buildDir string, configFile string, imageFile string,
	outputImageFile string, outputImageFormat string,
) error {
	var err error

	var config imagecustomizerapi.SystemConfig
	err = imagecustomizerapi.UnmarshalYamlFile(configFile, &config)
	if err != nil {
		return err
	}

	baseConfigPath, _ := filepath.Split(configFile)

	err = CustomizeImage(buildDir, baseConfigPath, &config, imageFile, outputImageFile, outputImageFormat)
	if err != nil {
		return err
	}

	return nil
}

func CustomizeImage(buildDir string, baseConfigPath string, config *imagecustomizerapi.SystemConfig, imageFile string,
	outputImageFile string, outputImageFormat string,
) error {
	var err error

	err = os.MkdirAll(buildDir, os.ModePerm)
	if err != nil {
		return err
	}

	// Convert image file to raw format, so that a kernel loop device can be used to make changes to the image.
	buildImageFile := filepath.Join(buildDir, "image.raw")

	_, _, err = shell.Execute("qemu-img", "convert", "-O", "raw", imageFile, buildImageFile)
	if err != nil {
		return fmt.Errorf("failed to convert image file to raw format: %w", err)
	}

	// Customize the raw image file.
	err = customizeImageHelper(buildDir, baseConfigPath, config, buildImageFile)
	if err != nil {
		return err
	}

	// Create final output image file.
	outDir := filepath.Dir(outputImageFile)
	os.MkdirAll(outDir, os.ModePerm)

	_, _, err = shell.Execute("qemu-img", "convert", "-O", toQemuImageFormat(outputImageFormat), buildImageFile, outputImageFile)
	if err != nil {
		return fmt.Errorf("failed to convert image file to format: %s: %w", outputImageFormat, err)
	}

	return nil
}

func toQemuImageFormat(imageFormat string) string {
	switch imageFormat {
	case "vhd":
		return "vpc"

	default:
		return imageFormat
	}
}

func customizeImageHelper(buildDir string, baseConfigPath string, config *imagecustomizerapi.SystemConfig,
	buildImageFile string,
) error {
	// Mount the raw disk image file.
	diskDevPath, err := diskutils.SetupLoopbackDevice(buildImageFile)
	if err != nil {
		return fmt.Errorf("failed to mount raw disk (%s) as a loopback device: %w", buildImageFile, err)
	}
	defer diskutils.DetachLoopbackDevice(diskDevPath)

	// Look for all the partitions on the image.
	newMountDirectories, mountPoints, err := findPartitions(buildDir, diskDevPath)
	if err != nil {
		return fmt.Errorf("failed to find disk partitions: %w", err)
	}

	// Create chroot environment.
	imageChrootDir := filepath.Join(buildDir, "imageroot")

	imageChroot := safechroot.NewChroot(imageChrootDir, false)
	err = imageChroot.Initialize("", newMountDirectories, mountPoints)
	if err != nil {
		return err
	}
	defer imageChroot.Close(false)

	// Do the actual customizations.
	err = doCustomizations(baseConfigPath, config, imageChroot)
	if err != nil {
		return err
	}

	return nil
}

func findPartitions(buildDir string, diskDevice string) ([]string, []*safechroot.MountPoint, error) {
	var err error

	diskPartitions, err := diskutils.GetDiskPartitions(diskDevice)
	if err != nil {
		return nil, nil, err
	}

	// Look for the boot partition (i.e. EFI system partition).
	var efiSystemPartition *diskutils.PartitionInfo
	for _, diskPartition := range diskPartitions {
		if diskPartition.PartitionTypeUuid == "c12a7328-f81f-11d2-ba4b-00a0c93ec93b" {
			efiSystemPartition = &diskPartition
			break
		}
	}

	if efiSystemPartition == nil {
		return nil, nil, fmt.Errorf("failed to find EFI system partition (%s)", diskDevice)
	}

	// Mount the boot partition.
	bootDir := filepath.Join(buildDir, "bootpartition")

	err = os.MkdirAll(bootDir, os.ModePerm)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create directory for EFI system partition: %w", err)
	}

	err = unix.Mount(efiSystemPartition.Path, bootDir, efiSystemPartition.FileSystemType, 0, "")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to mount EFI system partition: %w", err)
	}
	defer unix.Unmount(bootDir, 0)

	// Read the grub.cfg file.
	grubConfigFilePath := filepath.Join(bootDir, "boot/grub2/grub.cfg")
	grubConfigFile, err := os.ReadFile(grubConfigFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read grub.cfg file: %w", err)
	}

	// Look for the rootfs declaration line in the grub.cfg file.
	match := rootfsPartitionRegex.FindStringSubmatch(string(grubConfigFile))
	if match == nil {
		return nil, nil, fmt.Errorf("failed to find rootfs partition in grub.cfg file")
	}

	rootfsUuid := match[1]

	var rootfsPartition *diskutils.PartitionInfo
	for _, diskPartition := range diskPartitions {
		if diskPartition.Uuid == rootfsUuid {
			rootfsPartition = &diskPartition
			break
		}
	}

	// TODO: Read /etc/fstab file to find secondary partitions.
	mountPoints := []*safechroot.MountPoint{
		safechroot.NewPreDefaultsMountPoint(rootfsPartition.Path, "/", rootfsPartition.FileSystemType, 0, ""),
		safechroot.NewMountPoint(efiSystemPartition.Path, "/boot/efi", efiSystemPartition.FileSystemType, 0, ""),
	}

	return nil, mountPoints, nil
}

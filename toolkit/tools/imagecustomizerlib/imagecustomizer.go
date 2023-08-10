// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerlib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/imagecustomizerapi"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/imagegen/diskutils"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/safechroot"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/shell"
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

	imageCustomizerDir := filepath.Join(buildDir, "imagecustomizer")

	err = os.MkdirAll(imageCustomizerDir, os.ModePerm)
	if err != nil {
		return err
	}

	// Convert image file to raw format, so that a kernel loop device can be used to make changes to the image.
	buildImageFile := filepath.Join(imageCustomizerDir, "image.raw")

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
	newMountDirectories, mountPoints, err := findPartitions(diskDevPath)
	if err != nil {
		return err
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

func findPartitions(diskDevice string) ([]string, []*safechroot.MountPoint, error) {
	newMountDirectories := []string{}

	// TODO: Dynamically find partitions instead of hardcoding the mappings.
	mountPoints := []*safechroot.MountPoint{
		safechroot.NewPreDefaultsMountPoint(fmt.Sprintf("%sp2", diskDevice), "/", "ext4", 0, ""),
		safechroot.NewMountPoint(fmt.Sprintf("%sp1", diskDevice), "/boot", "vfat", 0, ""),
	}

	return newMountDirectories, mountPoints, nil
}

// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerlib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/imagecustomizerapi"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/file"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/logger"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/safechroot"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/safemount"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/shell"
	"golang.org/x/sys/unix"
)

const (
	tmpParitionDirName = "tmppartition"

	BaseImageName                = "image.raw"
	PartitionCustomizedImageName = "image2.raw"

	ImageRootDirName     = "imageroot"
	ToolsDirName         = "toolsroot"
	ImageRootInToolsPath = "/mnt/imageroot"
)

var (
	// Version specifies the version of the Mariner Image Customizer tool.
	// The value of this string is inserted during compilation via a linker flag.
	ToolVersion = ""
)

func CustomizeImageWithConfigFile(buildDir string, configFile string, imageFile string,
	rpmsSources []string, outputImageFile string, outputImageFormat string,
	useBaseImageRpmRepos bool, toolsBinPath string,
) error {
	var err error

	var config imagecustomizerapi.Config
	err = imagecustomizerapi.UnmarshalYamlFile(configFile, &config)
	if err != nil {
		return err
	}

	baseConfigPath, _ := filepath.Split(configFile)

	absBaseConfigPath, err := filepath.Abs(baseConfigPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of config file directory:\n%w", err)
	}

	err = CustomizeImage(buildDir, absBaseConfigPath, &config, imageFile, rpmsSources, outputImageFile,
		outputImageFormat, useBaseImageRpmRepos, toolsBinPath)
	if err != nil {
		return err
	}

	return nil
}

func CustomizeImage(buildDir string, baseConfigPath string, config *imagecustomizerapi.Config, imageFile string,
	rpmsSources []string, outputImageFile string, outputImageFormat string, useBaseImageRpmRepos bool,
	toolsBinPath string,
) error {
	var err error

	// Validate 'outputImageFormat' value.
	qemuOutputImageFormat, err := toQemuImageFormat(outputImageFormat)
	if err != nil {
		return err
	}

	// Validate config.
	err = validateConfig(baseConfigPath, config)
	if err != nil {
		return fmt.Errorf("invalid image config:\n%w", err)
	}

	// Normalize 'buildDir' path.
	buildDirAbs, err := filepath.Abs(buildDir)
	if err != nil {
		return err
	}

	// Create 'buildDir' directory.
	err = os.MkdirAll(buildDirAbs, os.ModePerm)
	if err != nil {
		return err
	}

	// Mount tools.
	toolsChroot := (*safechroot.Chroot)(nil)
	if toolsBinPath != "" {
		logger.Log.Infof("Mounting tools (%s)", toolsBinPath)

		toolsConnection := NewImageConnection()
		defer toolsConnection.Close()

		// Connect to squashfs file.
		err := toolsConnection.ConnectLoopback(toolsBinPath)
		if err != nil {
			return fmt.Errorf("failed to connect to tools bin (%s):\n%w", toolsBinPath, err)
		}

		toolsMountDir := filepath.Join(buildDirAbs, ToolsDirName)

		// Mount squashfs filesystem.
		mounts := []*safechroot.MountPoint{
			safechroot.NewPreDefaultsMountPoint(toolsConnection.Loopback().DevicePath(), "/", "squashfs", unix.MS_RDONLY, ""),
		}

		toolsConnection.ConnectChroot(toolsMountDir, false, nil, mounts)
		if err != nil {
			return fmt.Errorf("failed to mount tools bin (%s):\n%w", toolsBinPath, err)
		}

		toolsChroot = toolsConnection.Chroot()
	}

	// Convert image file to raw format, so that a kernel loop device can be used to make changes to the image.
	buildImageFile := filepath.Join(buildDirAbs, BaseImageName)

	logger.Log.Infof("Mounting base image: %s", buildImageFile)
	err = shell.ExecuteLiveWithErr(1, "qemu-img", "convert", "-O", "raw", imageFile, buildImageFile)
	if err != nil {
		return fmt.Errorf("failed to convert image file to raw format:\n%w", err)
	}

	// Customize the partitions.
	buildImageFile, err = customizePartitions(buildDirAbs, baseConfigPath, config, buildImageFile)
	if err != nil {
		return err
	}

	// Customize the raw image file.
	err = customizeImageHelper(buildDirAbs, baseConfigPath, config, buildImageFile, rpmsSources,
		useBaseImageRpmRepos, toolsChroot)
	if err != nil {
		return err
	}

	// Create final output image file.
	logger.Log.Infof("Writing: %s", outputImageFile)

	outDir := filepath.Dir(outputImageFile)
	os.MkdirAll(outDir, os.ModePerm)

	err = shell.ExecuteLiveWithErr(1, "qemu-img", "convert", "-O", qemuOutputImageFormat, buildImageFile,
		outputImageFile)
	if err != nil {
		return fmt.Errorf("failed to convert image file to format: %s:\n%w", outputImageFormat, err)
	}

	logger.Log.Infof("Success!")

	return nil
}

func toQemuImageFormat(imageFormat string) (string, error) {
	switch imageFormat {
	case "vhd":
		return "vpc", nil

	case "vhdx", "raw", "qcow2":
		return imageFormat, nil

	default:
		return "", fmt.Errorf("unsupported image format (supported: vhd, vhdx, raw, qcow2): %s", imageFormat)
	}
}

func validateConfig(baseConfigPath string, config *imagecustomizerapi.Config) error {
	var err error

	err = validateSystemConfig(baseConfigPath, &config.SystemConfig)
	if err != nil {
		return err
	}

	return nil
}

func validateSystemConfig(baseConfigPath string, config *imagecustomizerapi.SystemConfig) error {
	var err error

	for sourceFile := range config.AdditionalFiles {
		sourceFileFullPath := filepath.Join(baseConfigPath, sourceFile)
		isFile, err := file.IsFile(sourceFileFullPath)
		if err != nil {
			return fmt.Errorf("invalid AdditionalFiles source file (%s):\n%w", sourceFile, err)
		}

		if !isFile {
			return fmt.Errorf("invalid AdditionalFiles source file (%s): not a file", sourceFile)
		}
	}

	for i, script := range config.PostInstallScripts {
		err = validateScript(baseConfigPath, &script)
		if err != nil {
			return fmt.Errorf("invalid PostInstallScripts item at index %d: %w", i, err)
		}
	}

	for i, script := range config.FinalizeImageScripts {
		err = validateScript(baseConfigPath, &script)
		if err != nil {
			return fmt.Errorf("invalid FinalizeImageScripts item at index %d: %w", i, err)
		}
	}

	return nil
}

func validateScript(baseConfigPath string, script *imagecustomizerapi.Script) error {
	// Ensure that install scripts sit under the config file's parent directory.
	// This allows the install script to be run in the chroot environment by bind mounting the config directory.
	if !filepath.IsLocal(script.Path) {
		return fmt.Errorf("install script (%s) is not under config directory (%s)", script.Path, baseConfigPath)
	}

	// Verify that the file exists.
	fullPath := filepath.Join(baseConfigPath, script.Path)

	scriptStat, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("couldn't read install script (%s):\n%w", script.Path, err)
	}

	// Verify that the file has an executable bit set.
	if scriptStat.Mode()&0111 == 0 {
		return fmt.Errorf("install script (%s) does not have executable bit set", script.Path)
	}

	return nil
}

func customizeImageHelper(buildDir string, baseConfigPath string, config *imagecustomizerapi.Config,
	buildImageFile string, rpmsSources []string, useBaseImageRpmRepos bool, toolsChroot *safechroot.Chroot,
) error {
	// Connect to the image file.
	imageConnection, err := connectToExistingImage(buildImageFile, buildDir, ImageRootDirName)
	if err != nil {
		return err
	}
	defer imageConnection.Close()

	// Create a bind mount to the image's root in the tools directory.
	var imageInToolsMount *safemount.Mount
	if toolsChroot != nil {
		imagerootInToolsChrootDir := filepath.Join(toolsChroot.RootDir(), ImageRootInToolsPath)
		imageInToolsMount, err = safemount.NewMount(imageConnection.chroot.RootDir(), imagerootInToolsChrootDir, "",
			unix.MS_BIND|unix.MS_RDONLY, "", false)
		if err != nil {
			return err
		}
		defer imageInToolsMount.Close()
	}

	// Do the actual customizations.
	err = doCustomizations(buildDir, baseConfigPath, config, imageConnection.Chroot(), rpmsSources,
		useBaseImageRpmRepos, toolsChroot)
	if err != nil {
		return err
	}

	// Cleanup.
	if imageInToolsMount != nil {
		err = imageInToolsMount.CleanClose()
		if err != nil {
			return err
		}
	}

	err = imageConnection.CleanClose()
	if err != nil {
		return err
	}

	return nil
}

// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerlib

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/imagecustomizerapi"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/file"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/safechroot"
)

var (
	linuxCommandLineRegex = regexp.MustCompile(`\tlinux .* (\$kernelopts)`)
)

func doCustomizations(baseConfigPath string, config *imagecustomizerapi.SystemConfig, imageChroot *safechroot.Chroot) error {
	var err error

	err = updateHostname(config.Hostname, imageChroot)
	if err != nil {
		return err
	}

	err = copyAdditionalFiles(baseConfigPath, config.AdditionalFiles, imageChroot)
	if err != nil {
		return err
	}

	err = handleKernelCommandLine(config.KernelCommandLine.ExtraCommandLine, imageChroot)
	if err != nil {
		return fmt.Errorf("failed to add extra kernel command line: %w", err)
	}

	return nil
}

func updateHostname(hostname string, imageChroot *safechroot.Chroot) error {
	var err error

	if hostname == "" {
		return nil
	}

	hostnameFilePath := filepath.Join(imageChroot.RootDir(), "etc/hostname")
	err = file.Write(hostname, hostnameFilePath)
	if err != nil {
		return fmt.Errorf("failed to write hostname file: %w", err)
	}

	return nil
}

func copyAdditionalFiles(baseConfigPath string, additionalFiles map[string]imagecustomizerapi.FileConfigList, imageChroot *safechroot.Chroot) error {
	var err error

	for sourceFile, fileConfigs := range additionalFiles {
		for _, fileConfig := range fileConfigs {
			fileToCopy := safechroot.FileToCopy{
				Src:         filepath.Join(baseConfigPath, sourceFile),
				Dest:        fileConfig.Path,
				Permissions: (*fs.FileMode)(fileConfig.Permissions),
			}

			err = imageChroot.AddFiles(fileToCopy)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func handleKernelCommandLine(extraCommandLine string, imageChroot *safechroot.Chroot) error {
	var err error

	grub2ConfigFilePath := filepath.Join(imageChroot.RootDir(), "/boot/grub2/grub.cfg")

	// Read the existing grub.cfg file.
	grub2ConfigFileBytes, err := os.ReadFile(grub2ConfigFilePath)
	if err != nil {
		return fmt.Errorf("failed to read existing grub2 config file: %w", err)
	}

	grub2ConfigFile := string(grub2ConfigFileBytes)

	// Find the point where the new command line arguments should be added.
	match := linuxCommandLineRegex.FindStringSubmatchIndex(grub2ConfigFile)
	if match == nil {
		return fmt.Errorf("failed to find Linux kernel command line params in grub2 config file")
	}

	insertIndex := match[1]

	// Insert new command line arguments.
	newGrub2ConfigFile := grub2ConfigFile[:insertIndex] + extraCommandLine + " " + grub2ConfigFile[insertIndex:]

	// Update grub.cfg file.
	err = os.WriteFile(grub2ConfigFilePath, []byte(newGrub2ConfigFile), 0)
	if err != nil {
		return fmt.Errorf("failed to write new grub2 config file: %w", err)
	}

	return nil
}

// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerlib

import (
	"fmt"
	"path"
	"strings"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/imagecustomizerapi"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/logger"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/safechroot"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/shell"
)

const (
	rpmsMountParentDirInChroot = "/sourcerpms"
)

func updatePackages(buildDir string, baseConfigPath string, packageLists []string, packages []string,
	imageChroot *safechroot.Chroot, rpmsSources []string,
) error {
	var err error

	allPackages, err := collectPackagesList(baseConfigPath, packageLists, packages)
	if err != nil {
		return err
	}

	err = updatePackagesHelper(buildDir, allPackages, imageChroot, rpmsSources)
	if err != nil {
		return err
	}

	return nil
}

func collectPackagesList(baseConfigPath string, packageLists []string, packages []string) ([]string, error) {
	var err error

	// Read in the packages from the package list files.
	var allPackages []string
	for _, packageListRelativePath := range packageLists {
		packageListFilePath := path.Join(baseConfigPath, packageListRelativePath)

		var packageList imagecustomizerapi.PackageList
		err = imagecustomizerapi.UnmarshalYamlFile(packageListFilePath, &packageList)
		if err != nil {
			return nil, fmt.Errorf("failed to read package list file (%s): %w", packageListFilePath, err)
		}

		allPackages = append(allPackages, packageList.Packages...)
	}

	allPackages = append(allPackages, packages...)
	return allPackages, nil
}

func updatePackagesHelper(buildDir string, packages []string, imageChroot *safechroot.Chroot, rpmsSources []string) error {
	var err error

	if len(packages) <= 0 {
		return nil
	}

	if len(rpmsSources) <= 0 {
		return fmt.Errorf("have %d packages to install but no RPM sources were specified", len(packages))
	}

	// Mount RPM sources.
	mounts, err := mountRpmSources(buildDir, imageChroot, rpmsSources)
	if err != nil {
		return err
	}
	defer mounts.close()

	// Create tdnf command args.
	// Note: When using `--repofromdir`, tdnf will not use any default repos and will only use the last
	// `--repofromdir` specified.
	tnfInstallCommonArgs := []string{
		"-v", "install", "--nogpgcheck", "--assumeyes",
		fmt.Sprintf("--repofromdir=sourcerpms,%s", rpmsMountParentDirInChroot),
	}

	// Add placeholder arg for the package name.
	tnfInstallCommonArgs = append(tnfInstallCommonArgs, "")

	// Install packages.
	// Do this one at a time, to avoid running out of memory.
	for _, packageName := range packages {
		tnfInstallCommonArgs[len(tnfInstallCommonArgs)-1] = packageName

		err = imageChroot.Run(func() error {
			err := shell.ExecuteLiveWithCallback(tdnfInstallStdoutFilter, logger.Log.Warn, false, "tdnf", tnfInstallCommonArgs...)
			return err
		})
		if err != nil {
			return fmt.Errorf("failed to install package (%s): %w", packageName, err)
		}
	}

	// Unmount RPM sources.
	err = mounts.close()
	if err != nil {
		return err
	}

	return nil
}

// Process the stdout of a `tdnf install -v` call and send the list of installed packages to the debug log.
func tdnfInstallStdoutFilter(args ...interface{}) {
	const tdnfInstallPrefix = "Installing/Updating: "

	if len(args) == 0 {
		return
	}

	line := args[0].(string)
	if !strings.HasPrefix(line, tdnfInstallPrefix) {
		return
	}

	logger.Log.Debug(line)
}

func extractTarball(tarballFile string, directory string) error {
	var err error

	_, stderr, err := shell.Execute("tar", "-xf", tarballFile, "-C", directory)
	if err != nil {
		logger.Log.Debugf("tar stderr: %s", stderr)
		return err
	}

	return nil
}

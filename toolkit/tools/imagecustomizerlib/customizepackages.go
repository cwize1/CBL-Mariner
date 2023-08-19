// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerlib

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/imagecustomizerapi"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/file"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/logger"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/safechroot"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/safemount.go"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/shell"
	"golang.org/x/sys/unix"
)

func updatePackages(buildDir string, baseConfigPath string, packageLists []string, packages []string,
	imageChroot *safechroot.Chroot, rpmsSources []string,
) error {
	var err error

	// Read in the packages from the package list files.
	var allPackages []string
	for _, packageListRelativePath := range packageLists {
		packageListFilePath := path.Join(baseConfigPath, packageListRelativePath)

		var packageList imagecustomizerapi.PackageList
		err = imagecustomizerapi.UnmarshalYamlFile(packageListFilePath, &packageList)
		if err != nil {
			return fmt.Errorf("failed to read package list file (%s): %w", packageListFilePath, err)
		}

		allPackages = append(allPackages, packageList.Packages...)
	}

	allPackages = append(allPackages, packages...)
	err = updatePackagesHelper(buildDir, allPackages, imageChroot, rpmsSources)
	if err != nil {
		return err
	}

	return nil
}

func updatePackagesHelper(buildDir string, packages []string, imageChroot *safechroot.Chroot, rpmsSources []string) error {
	var err error

	if len(packages) <= 0 {
		return nil
	}

	if len(rpmsSources) <= 0 {
		return fmt.Errorf("have %d packages to install but no RPM sources were specified", len(packages))
	}

	extractedRpmsDir := path.Join(buildDir, "extracted_rpms")
	rpmsMountParentDirChroot := "/sourcerpms"
	rpmsMountParentDir := path.Join(imageChroot.RootDir(), rpmsMountParentDirChroot)

	// Create temporary directory for RPM sources to be mounted (and fail if it already exists).
	err = os.Mkdir(rpmsMountParentDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create source rpms directory (%s): %w", rpmsMountParentDir, err)
	}

	// Mount the RPM sources.
	var mounts []*safemount.Mount

	for i, rpmSource := range rpmsSources {
		rpmSourceIsFile, err := file.IsFile(rpmSource)
		if err != nil {
			return fmt.Errorf("failed to get file type of RPM source (%s): %w", rpmSource, err)
		}

		var rpmSourceName string
		var rpmsDirectory string

		// We assume RPM sources that are files are RPM tarballs.
		if rpmSourceIsFile {
			// Get a unique ID for the RPM tarball.
			logger.Log.Debugf("Calculating SHA-256 of rpms tarball (%s)", rpmSource)
			rpmSourceHash, err := file.GenerateSHA256(rpmSource)
			if err != nil {
				return fmt.Errorf("failed to get hash of RPM tarball (%s): %w", rpmSource, err)
			}

			// Check if the tarball has already been extracted.
			extractDirectory := path.Join(extractedRpmsDir, rpmSourceHash)
			extractDirectoryExists, err := file.DirExists(extractDirectory)
			if err != nil {
				return fmt.Errorf("failed to stat tarball extract directory (%s): %w", extractDirectory, err)
			}

			if !extractDirectoryExists {
				err = os.MkdirAll(extractDirectory, os.ModePerm)
				if err != nil {
					return fmt.Errorf("failed to create RPMs extract directory (%s): %w", extractedRpmsDir, err)
				}

				// Extract the RPMs tarball.
				logger.Log.Debugf("Extracting rpms tarball (%s)", rpmSource)
				err = extractTarball(rpmSource, extractDirectory)
				if err != nil {
					removeErr := os.RemoveAll(extractDirectory)
					if removeErr != nil {
						logger.Log.Warnf("failed to delete tarball extract directory (%s)", extractDirectory)
					}
					return fmt.Errorf("failed to extract tarball (%s): %w", rpmSource, err)
				}
			}

			rpmSourceName = path.Base(rpmSource)
			if extensionIndex := strings.Index(rpmSourceName, "."); extensionIndex >= 0 {
				rpmSourceName = rpmSourceName[:extensionIndex]
			}

			rpmsDirectory = extractDirectory
		} else {
			rpmSourceName = path.Base(rpmSource)
			rpmsDirectory = rpmSource
		}

		targetName := fmt.Sprintf("%02d%s", i, rpmSourceName)
		mountTargetDirectory := path.Join(imageChroot.RootDir(), rpmsMountParentDirChroot, targetName)

		// Create a read-only bind mount.
		mount, err := safemount.NewMount(rpmsDirectory, mountTargetDirectory, "", unix.MS_BIND|unix.MS_RDONLY, "")
		if err != nil {
			return fmt.Errorf("failed to mount RPM source directory from (%s) to (%s): %w", rpmsDirectory, mountTargetDirectory, err)
		}
		defer mount.Close()

		mounts = append(mounts, mount)
	}

	// Create tdnf command args.
	// Note: When using `--repofromdir`, tdnf will not use any default repos and will only use the last
	// `--repofromdir` specified.
	tnfInstallCommonArgs := []string{
		"-v", "install", "--nogpgcheck", "--assumeyes",
		fmt.Sprintf("--repofromdir=sourcerpms,%s", rpmsMountParentDirChroot),
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

	// Unmount rpm source directories.
	for _, mount := range mounts {
		err = mount.Close()
		if err != nil {
			return fmt.Errorf("failed to unmount (%s): %w", mount.Target(), err)
		}

		err = os.Remove(mount.Target())
		if err != nil {
			return fmt.Errorf("failed to delete source rpms mount directory (%s): %w", rpmsMountParentDir, err)
		}
	}

	// Delete the temporary directory.
	err = os.Remove(rpmsMountParentDir)
	if err != nil {
		return fmt.Errorf("failed to delete source rpms directory (%s): %w", rpmsMountParentDir, err)
	}

	return nil
}

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

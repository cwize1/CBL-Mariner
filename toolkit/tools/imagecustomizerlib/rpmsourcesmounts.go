// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerlib

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/file"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/logger"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/safechroot"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/safemount.go"
	"golang.org/x/sys/unix"
)

type rpmSourcesMounts struct {
	rpmsMountParentDir        string
	rpmsMountParentDirCreated bool
	mounts                    []*safemount.Mount
}

func mountRpmSources(buildDir string, imageChroot *safechroot.Chroot, rpmsSources []string) (*rpmSourcesMounts, error) {
	var err error

	var mounts rpmSourcesMounts
	err = mounts.mountRpmSourcesHelper(buildDir, imageChroot, rpmsSources)
	if err != nil {
		cleanupErr := mounts.close()
		if cleanupErr != nil {
			logger.Log.Warnf("rpm sources mount cleanup failed: %s", cleanupErr)
		}
		return nil, err
	}

	return &mounts, nil
}

func (m *rpmSourcesMounts) mountRpmSourcesHelper(buildDir string, imageChroot *safechroot.Chroot, rpmsSources []string) error {
	var err error

	extractedRpmsDir := path.Join(buildDir, "extracted_rpms")
	m.rpmsMountParentDir = path.Join(imageChroot.RootDir(), rpmsMountParentDirInChroot)

	// Create temporary directory for RPM sources to be mounted (and fail if it already exists).
	err = os.Mkdir(m.rpmsMountParentDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create source rpms directory (%s): %w", m.rpmsMountParentDir, err)
	}

	m.rpmsMountParentDirCreated = true

	// Mount the RPM sources.
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

			// Get the name of the tarball file, without the extension.
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
		mountTargetDirectory := path.Join(imageChroot.RootDir(), rpmsMountParentDirInChroot, targetName)

		// Create a read-only bind mount.
		mount, err := safemount.NewMount(rpmsDirectory, mountTargetDirectory, "", unix.MS_BIND|unix.MS_RDONLY, "", true)
		if err != nil {
			return fmt.Errorf("failed to mount RPM source directory from (%s) to (%s): %w", rpmsDirectory, mountTargetDirectory, err)
		}

		m.mounts = append(m.mounts, mount)
	}

	return nil
}

func (m *rpmSourcesMounts) close() error {
	var err error
	var errs []error

	// Unmount rpm source directories.
	for _, mount := range m.mounts {
		err = mount.Close()
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}

	if len(errs) > 0 {
		err = errors.Join(errs...)
		err = fmt.Errorf("failed to cleanup RPM sources mounts:\n%w", err)
		return err
	}

	// Delete the temporary directory.
	if m.rpmsMountParentDirCreated {
		// Note: Do not use `RemoveAll` here in case there are any leftover mounts that failed to unmount.
		err = os.Remove(m.rpmsMountParentDir)
		if err != nil {
			return fmt.Errorf("failed to delete source rpms directory (%s): %w", m.rpmsMountParentDir, err)
		}

		m.rpmsMountParentDirCreated = false
	}

	return nil
}

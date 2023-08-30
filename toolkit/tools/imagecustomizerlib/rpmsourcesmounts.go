// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerlib

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/file"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/logger"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/packagerepo/repomanager/rpmrepomanager"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/safechroot"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/safemount.go"
	"golang.org/x/sys/unix"
	"gopkg.in/ini.v1"
)

const (
	rpmsMountParentDirInChroot = "/_localrpms"
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

	allReposConfig := ini.Empty()

	// Mount the RPM sources.
	for _, rpmSource := range rpmsSources {
		fileType, err := getRpmSourceFileType(rpmSource)
		if err != nil {
			return fmt.Errorf("failed to get RPM source file type (%s): %w", rpmSource, err)
		}

		switch fileType {
		case "dir":
			err = m.createRepoFromDirectory(rpmSource, allReposConfig, imageChroot)

		case "gz":
			err = m.createRepoFromRpmsTarball(extractedRpmsDir, rpmSource, allReposConfig, imageChroot)

		case "repo.conf":
			err = m.createRepoFromRepoConfig(rpmSource, allReposConfig, imageChroot)

		default:
			return fmt.Errorf("unknown RPM source type (%s)", rpmSource)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *rpmSourcesMounts) createRepoFromDirectory(rpmSource string, allReposConfig *ini.File,
	imageChroot *safechroot.Chroot,
) error {
	// Turn directory into an RPM repo.
	err := rpmrepomanager.CreateOrUpdateRepo(rpmSource)
	if err != nil {
		return fmt.Errorf("failed create RPMs repo from directory (%s): %w", rpmSource, err)
	}

	rpmSourceName := path.Base(rpmSource)

	// Mount the directory.
	err = m.mountRpmsDirectory(rpmSourceName, rpmSource, imageChroot)
	if err != nil {
		return err
	}

	return nil
}

// Creates an RPM repo from a tarball containing *.rpm files.
func (m *rpmSourcesMounts) createRepoFromRpmsTarball(extractedRpmsDir string, rpmSource string,
	allReposConfig *ini.File, imageChroot *safechroot.Chroot,
) error {
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

		err = createRepoFromRpmsTarballHelper(rpmSource, extractDirectory)
		if err != nil {
			removeErr := os.RemoveAll(extractDirectory)
			if removeErr != nil {
				logger.Log.Warnf("failed to delete tarball extract directory (%s)", extractDirectory)
			}
			return err
		}

	}

	// Get the name of the tarball file, without the extension.
	rpmSourceName := path.Base(rpmSource)
	if extensionIndex := strings.Index(rpmSourceName, "."); extensionIndex >= 0 {
		rpmSourceName = rpmSourceName[:extensionIndex]
	}

	// Mount the directory.
	err = m.mountRpmsDirectory(rpmSourceName, extractDirectory, imageChroot)
	if err != nil {
		return err
	}

	return nil
}

// Extract the RPMs tarball and then turn the directory into an RPM repo.
func createRepoFromRpmsTarballHelper(rpmSource string, extractDirectory string) error {
	var err error

	// Extract the RPMs tarball.
	logger.Log.Debugf("Extracting rpms tarball (%s)", rpmSource)
	err = extractTarball(rpmSource, extractDirectory)
	if err != nil {
		return fmt.Errorf("failed to extract RPMs tarball (%s): %w", rpmSource, err)
	}

	// Turn directory into an RPM repo.
	err = rpmrepomanager.CreateRepo(extractDirectory)
	if err != nil {
		return fmt.Errorf("failed create RPMs repo from RPMs tarball (%s): %w", rpmSource, err)
	}

	return nil
}

func (m *rpmSourcesMounts) createRepoFromRepoConfig(rpmSource string, allReposConfig *ini.File,
	imageChroot *safechroot.Chroot,
) error {
	// Parse the repo config file.
	reposConfig, err := ini.Load(rpmSource)
	if err != nil {
		return fmt.Errorf("failed load repo config file (%s): %w", rpmSource, err)
	}

	for _, repoConfig := range reposConfig.Sections() {
		if repoConfig.Name() == "" {
			return fmt.Errorf("rpm repo config files must not contain nameless sections (%s)", rpmSource)
		}

		// Copy over the repo details to the all-repos config.
		newSection, err := allReposConfig.NewSection(repoConfig.Name())
		if err != nil {
			return err
		}

		for _, key := range repoConfig.Keys() {
			_, err := newSection.NewKey(key.Name(), key.Value())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *rpmSourcesMounts) mountRpmsDirectory(rpmSourceName string, rpmsDirectory string,
	imageChroot *safechroot.Chroot,
) error {
	i := len(m.mounts)
	targetName := fmt.Sprintf("%02d%s", i, rpmSourceName)
	mountTargetDirectory := path.Join(imageChroot.RootDir(), rpmsMountParentDirInChroot, targetName)

	// Create a read-only bind mount.
	mount, err := safemount.NewMount(rpmsDirectory, mountTargetDirectory, "", unix.MS_BIND|unix.MS_RDONLY, "", true)
	if err != nil {
		return fmt.Errorf("failed to mount RPM source directory from (%s) to (%s): %w", rpmsDirectory, mountTargetDirectory, err)
	}

	m.mounts = append(m.mounts, mount)
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

func getRpmSourceFileType(filePath string) (string, error) {
	// First, check if path points to a directory.
	isDir, err := file.IsDir(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get type of RPM source (%s): %w", filePath, err)
	}

	if isDir {
		return "dir", nil
	}

	// Check the file's signature.
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0)
	if err != nil {
		return "", err
	}
	defer file.Close()

	firstBytes := make([]byte, 2)
	readByteCount, err := file.Read(firstBytes)
	if err != nil {
		return "", err
	}

	switch {
	case readByteCount >= 2 && bytes.Equal(firstBytes[:2], []byte{0x1F, 0x8B}):
		return "gz", nil

	case filepath.Ext(filePath) == "conf":
		return "repo.conf", nil

	default:
		return "", nil
	}
}

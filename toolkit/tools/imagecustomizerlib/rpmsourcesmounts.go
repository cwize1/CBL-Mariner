// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerlib

import (
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
	"github.com/sirupsen/logrus"
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
	allReposConfigFilePath    string
}

func mountRpmSources(buildDir string, imageChroot *safechroot.Chroot, rpmsSources []string,
	useBaseImageRpmRepos bool,
) (*rpmSourcesMounts, error) {
	var err error

	var mounts rpmSourcesMounts
	err = mounts.mountRpmSourcesHelper(buildDir, imageChroot, rpmsSources, useBaseImageRpmRepos)
	if err != nil {
		cleanupErr := mounts.close()
		if cleanupErr != nil {
			logger.Log.Warnf("rpm sources mount cleanup failed: %s", cleanupErr)
		}
		return nil, err
	}

	return &mounts, nil
}

func (m *rpmSourcesMounts) mountRpmSourcesHelper(buildDir string, imageChroot *safechroot.Chroot, rpmsSources []string,
	useBaseImageRpmRepos bool,
) error {
	var err error

	extractedRpmsDir := path.Join(buildDir, "extracted_rpms")
	m.rpmsMountParentDir = path.Join(imageChroot.RootDir(), rpmsMountParentDirInChroot)

	// Create temporary directory for RPM sources to be mounted (and fail if it already exists).
	err = os.Mkdir(m.rpmsMountParentDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create source rpms directory (%s): %w", m.rpmsMountParentDir, err)
	}

	m.rpmsMountParentDirCreated = true

	// Bind mount the resolv.conf file, so that the chroot has internet access.
	err = m.mountResolvConf(imageChroot)
	if err != nil {
		return err
	}

	// Unfortunatley, tdnf doesn't support the repository priority field.
	// So, to ensure repos are used in the correct order, create a single config file containing all the repos, specified
	// in the order of highest priority to lowest priority.
	allReposConfig := ini.Empty()

	// Include base image's RPM sources.
	if useBaseImageRpmRepos {
		reposPath := filepath.Join(imageChroot.RootDir(), "/etc/yum.repos.d")
		entries, err := os.ReadDir(reposPath)
		if err != nil {
			return fmt.Errorf("failed to read base image's repos directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			if !strings.HasSuffix(name, ".repo") {
				continue
			}

			repoFilePath := filepath.Join(reposPath, name)
			err = m.createRepoFromRepoConfig(repoFilePath, false, allReposConfig, imageChroot)
			if err != nil {
				return fmt.Errorf("failed to add base image's repo (%s): %w", name, err)
			}
		}
	}

	// Mount the RPM sources.
	for _, rpmSource := range rpmsSources {
		fileType, err := getRpmSourceFileType(rpmSource)
		if err != nil {
			return fmt.Errorf("failed to get RPM source file type (%s): %w", rpmSource, err)
		}

		switch fileType {
		case "dir":
			err = m.createRepoFromDirectory(rpmSource, allReposConfig, imageChroot)

		case "tar":
			err = m.createRepoFromRpmsTarball(extractedRpmsDir, rpmSource, allReposConfig, imageChroot)

		case "conf":
			err = m.createRepoFromRepoConfig(rpmSource, true, allReposConfig, imageChroot)

		default:
			return fmt.Errorf("unknown RPM source type (%s)", rpmSource)
		}
		if err != nil {
			return err
		}
	}

	// Create all-repos config file.
	m.allReposConfigFilePath = filepath.Join(imageChroot.RootDir(), rpmsMountParentDirInChroot, "allrepos.repo")
	logger.Log.Debugf("Writing allrepos.repo (%s)", m.allReposConfigFilePath)

	err = allReposConfig.SaveTo(m.allReposConfigFilePath)
	if err != nil {
		return fmt.Errorf("failed to save all-repos config file (%s): %w", m.allReposConfigFilePath, err)
	}

	if logger.Log.IsLevelEnabled(logrus.TraceLevel) {
		allReposConfigString, err := os.ReadFile(m.allReposConfigFilePath)
		if err == nil {
			logger.Log.Tracef("allrepos.repo:\n%s", allReposConfigString)
		}
	}

	return nil
}

func (m *rpmSourcesMounts) mountResolvConf(imageChroot *safechroot.Chroot) error {
	resolvConfInChroot := filepath.Join(imageChroot.RootDir(), "/etc/resolv.conf")

	// Create a read-only bind mount for the resolv.conf file.
	mount, err := safemount.NewMount("/etc/resolv.conf", resolvConfInChroot, "", unix.MS_BIND|unix.MS_RDONLY, "", true)
	if err != nil {
		return fmt.Errorf("failed to bind mount resolv.conf file: %w", err)
	}

	m.mounts = append(m.mounts, mount)
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
	mountTargetDirectoryInChroot, err := m.mountRpmsDirectory(rpmSourceName, rpmSource, imageChroot)
	if err != nil {
		return err
	}

	// Add local repo config.
	err = appendLocalRepo(allReposConfig, mountTargetDirectoryInChroot)
	if err != nil {
		return fmt.Errorf("failed to append local repo config: %w", err)
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
	mountTargetDirectoryInChroot, err := m.mountRpmsDirectory(rpmSourceName, extractDirectory, imageChroot)
	if err != nil {
		return err
	}

	// Add local repo config.
	err = appendLocalRepo(allReposConfig, mountTargetDirectoryInChroot)
	if err != nil {
		return fmt.Errorf("failed to append local repo config: %w", err)
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

func (m *rpmSourcesMounts) createRepoFromRepoConfig(rpmSource string, isHostConfig bool, allReposConfig *ini.File,
	imageChroot *safechroot.Chroot,
) error {
	// Parse the repo config file.
	reposConfig, err := ini.Load(rpmSource)
	if err != nil {
		return fmt.Errorf("failed load repo config file (%s): %w", rpmSource, err)
	}

	// Iterate through the list of repos.
	for _, repoConfig := range reposConfig.Sections() {
		if repoConfig.Name() == "" {
			return fmt.Errorf("rpm repo config files must not contain nameless sections (%s)", rpmSource)
		}

		if isHostConfig {
			// Check if the repo points to a local directory.
			baseurl := repoConfig.Key("baseurl").String()
			filePath, hasFilePrefix := strings.CutPrefix(baseurl, "file://")
			if hasFilePrefix {
				// Mount the directory in the chroot.
				rpmSourceName := path.Base(baseurl)
				mountTargetDirectoryInChroot, err := m.mountRpmsDirectory(rpmSourceName, filePath, imageChroot)
				if err != nil {
					return fmt.Errorf("failed mount repo config local directory (%s): %w", rpmSource, err)
				}

				// Change the baseurl to point to the bind mount directory.
				newBaseurl := fmt.Sprintf("file://%s", mountTargetDirectoryInChroot)
				repoConfig.Key("baseurl").SetValue(newBaseurl)
			}
		}

		// Copy over the repo details to the all-repos config.
		err := appendIniSection(allReposConfig, repoConfig)
		if err != nil {
			return fmt.Errorf("failed to append repo config: %w", err)
		}
	}

	return nil
}

func (m *rpmSourcesMounts) mountRpmsDirectory(rpmSourceName string, rpmsDirectory string,
	imageChroot *safechroot.Chroot,
) (string, error) {
	i := len(m.mounts)
	targetName := fmt.Sprintf("%02d%s", i, rpmSourceName)
	mountTargetDirectoryInChroot := path.Join(rpmsMountParentDirInChroot, targetName)
	mountTargetDirectory := path.Join(imageChroot.RootDir(), mountTargetDirectoryInChroot)

	// Create a read-only bind mount.
	mount, err := safemount.NewMount(rpmsDirectory, mountTargetDirectory, "", unix.MS_BIND|unix.MS_RDONLY, "", true)
	if err != nil {
		return "", fmt.Errorf("failed to mount RPM source directory from (%s) to (%s): %w", rpmsDirectory, mountTargetDirectory, err)
	}

	m.mounts = append(m.mounts, mount)
	return mountTargetDirectoryInChroot, nil
}

func (m *rpmSourcesMounts) close() error {
	var err error
	var errs []error

	// Delete allrepos.repo file (if it exists).
	err = os.RemoveAll(m.allReposConfigFilePath)
	if err != nil {
		errs = append(errs, err)
	}

	// Unmount rpm source directories.
	for _, mount := range m.mounts {
		err = mount.Close()
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}

	// Join all the errors together.
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

func getRpmSourceFileType(rpmSourcePath string) (string, error) {
	// First, check if path points to a directory.
	isDir, err := file.IsDir(rpmSourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to get type of RPM source (%s): %w", rpmSourcePath, err)
	}

	if isDir {
		return "dir", nil
	}

	filename := filepath.Base(rpmSourcePath)
	dotIndex := strings.Index(filename, ".")
	fileExt := ""
	if dotIndex >= 0 {
		fileExt = filename[dotIndex:]
	}

	switch fileExt {
	case ".tar", ".tar.gz":
		return "tar", nil

	case ".conf":
		return "conf", nil

	default:
		return "", nil
	}
}

func appendLocalRepo(iniFile *ini.File, mountTargetDirectoryInChroot string) error {
	repoName := filepath.Base(mountTargetDirectoryInChroot)
	iniSection, err := iniFile.NewSection(repoName)
	if err != nil {
		return err
	}

	_, err = iniSection.NewKey("name", repoName)
	if err != nil {
		return err
	}

	baseurl := fmt.Sprintf("file://%s", mountTargetDirectoryInChroot)

	_, err = iniSection.NewKey("baseurl", baseurl)
	if err != nil {
		return err
	}

	_, err = iniSection.NewKey("enabled", "1")
	if err != nil {
		return err
	}

	return nil
}

func appendIniSection(iniFile *ini.File, iniSection *ini.Section) error {
	newSection, err := iniFile.NewSection(iniSection.Name())
	if err != nil {
		return err
	}

	for _, key := range iniSection.Keys() {
		_, err := newSection.NewKey(key.Name(), key.Value())
		if err != nil {
			return err
		}
	}

	return nil
}

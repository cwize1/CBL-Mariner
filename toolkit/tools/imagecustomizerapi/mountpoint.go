// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"fmt"
	"path"
)

// MountPoint holds the mounting information for each partition.
type MountPoint struct {
	// ID is used to correlate `Partition` objects with `MountPoint` objects.
	ID string `yaml:"id"`
	// FilesystemType is the type of file system to use on the partition.
	FilesystemType FileSystemType `yaml:"fsType"`
	// MountIdentifierType is how the source block device is referenced.
	MountIdentifierType MountIdentifierType `yaml:"mountIdentifierType"`
	// Options is the extra options for the mount.
	Options string `yaml:"options"`
	// Path is the target directory for the mount.
	Path string `yaml:"path"`
}

// IsValid returns an error if the PartitionSetting is not valid
func (p *MountPoint) IsValid() error {
	err := p.FilesystemType.IsValid()
	if err != nil {
		return fmt.Errorf("invalid MountPoint (%s) FilesystemType value:\n%w", p.ID, err)
	}

	err = p.MountIdentifierType.IsValid()
	if err != nil {
		return err
	}

	if p.Path != "" && !path.IsAbs(p.Path) {
		return fmt.Errorf("target path (%s) must be an absolute path", p.Path)
	}

	return nil
}

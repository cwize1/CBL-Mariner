// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"fmt"
	"path"
)

// MountPoint holds the mounting information for each partition.
type MountPoint struct {
	DeviceId            string              `yaml:"deviceId"`
	MountIdentifierType MountIdentifierType `yaml:"mountIdentifierType"`
	Options             string              `yaml:"options"`
	Path                string              `yaml:"path"`
}

// IsValid returns an error if the PartitionSetting is not valid
func (p *MountPoint) IsValid() error {
	err := p.MountIdentifierType.IsValid()
	if err != nil {
		return err
	}

	if p.Path != "" && !path.IsAbs(p.Path) {
		return fmt.Errorf("target path (%s) must be an absolute path", p.Path)
	}

	return nil
}

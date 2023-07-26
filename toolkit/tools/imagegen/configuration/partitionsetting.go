// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

// Parser for the image builder's configuration schemas.

package configuration

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// PartitionSetting holds the mounting information for each partition.
type PartitionSetting struct {
	RemoveDocs       bool            `json:"RemoveDocs" yaml:"RemoveDocs"`
	ID               string          `json:"ID" yaml:"ID"`
	MountIdentifier  MountIdentifier `json:"MountIdentifier" yaml:"MountIdentifier"`
	MountOptions     string          `json:"MountOptions" yaml:"MountOptions"`
	MountPoint       string          `json:"MountPoint" yaml:"MountPoint"`
	OverlayBaseImage string          `json:"OverlayBaseImage" yaml:"OverlayBaseImage"`
	RdiffBaseImage   string          `json:"RdiffBaseImage" yaml:"RdiffBaseImage"`
}

var defaultPartitionSetting PartitionSetting = PartitionSetting{
	MountIdentifier: GetDefaultMountIdentifier(),
}

// GetDefaultPartitionSetting returns a copy of the default partition setting
func GetDefaultPartitionSetting() (defaultVal PartitionSetting) {
	defaultVal = defaultPartitionSetting
	return defaultVal
}

// IsValid returns an error if the PartitionSetting is not valid
func (p *PartitionSetting) IsValid() (err error) {
	return nil
}

// UnmarshalYAML Unmarshals a PartitionSetting entry
func (p *PartitionSetting) UnmarshalYAML(value *yaml.Node) (err error) {
	// Use an intermediate type which will use the default JSON unmarshal implementation
	type IntermediateTypePartitionSetting PartitionSetting

	// Populate non-standard default values
	*p = GetDefaultPartitionSetting()

	err = value.Decode((*IntermediateTypePartitionSetting)(p))
	if err != nil {
		return fmt.Errorf("failed to parse [PartitionSetting]: %w", err)
	}

	// Now validate the resulting unmarshaled object
	err = p.IsValid()
	if err != nil {
		return fmt.Errorf("failed to parse [PartitionSetting]: %w", err)
	}
	return
}

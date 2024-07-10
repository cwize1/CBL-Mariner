// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

type Overlay struct {
	Lower string `yaml:"lower"`
	Upper string `yaml:"upper"`
	Work  string `yaml:"work"`
	// The additional options for the mount.
	Options string `yaml:"options"`
	// The target directory path of the mount.
	Target string `yaml:"target"`
}

func (o *Overlay) IsValid() error {
	return nil
}

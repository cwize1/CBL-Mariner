// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

// OSConfig defines how each system present on the image is supposed to be configured.
type Packages struct {
	UpdateExistingPackages bool     `yaml:"updateExistingPackages"`
	InstallLists           []string `yaml:"installLists"`
	Install                []string `yaml:"install"`
	RemoveLists            []string `yaml:"removeLists"`
	Remove                 []string `yaml:"remove"`
	UpdateLists            []string `yaml:"updateLists"`
	Update                 []string `yaml:"update"`
}

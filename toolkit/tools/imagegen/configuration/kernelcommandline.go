// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

// Parser for the image builder's configuration schemas.

package configuration

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// KernelCommandLine holds extra command line parameters which can be
// added to the grub config file.
// - ImaPolicy: A list of IMA policies which will be used together
// - ExtraCommandLine: Arbitrary parameters which will be appended to the
//   end of the kernel command line
type KernelCommandLine struct {
	ImaPolicy        []ImaPolicy `json:"ImaPolicy" yaml:"ImaPolicy"`
	SELinux          SELinux     `json:"SELinux" yaml:"SELinux"`
	ExtraCommandLine string      `json:"ExtraCommandLine" yaml:"ExtraCommandLine"`
}

// GetSedDelimeter returns the delimeter which should be used with sed
// to find/replace the command line strings.
func (k *KernelCommandLine) GetSedDelimeter() (delimeter string) {
	return "`"
}

// IsValid returns an error if the KernelCommandLine is not valid
func (k *KernelCommandLine) IsValid() (err error) {
	for _, ima := range k.ImaPolicy {
		if err = ima.IsValid(); err != nil {
			return
		}
	}

	err = k.SELinux.IsValid()
	if err != nil {
		return err
	}

	// A character needs to be set aside for use as the sed delimiter, make sure it isn't included in the provided string
	if strings.Contains(k.ExtraCommandLine, k.GetSedDelimeter()) {
		return fmt.Errorf("ExtraCommandLine contains character %s which is reserved for use by sed", k.GetSedDelimeter())
	}

	return
}

// UnmarshalYAML unmarshals a KernelCommandLine entry
func (k *KernelCommandLine) UnmarshalYAML(value *yaml.Node) (err error) {
	// Use an intermediate type which will use the default JSON unmarshal implementation
	type IntermediateTypeKernelCommandLine KernelCommandLine
	err = value.Decode((*IntermediateTypeKernelCommandLine)(k))
	if err != nil {
		return fmt.Errorf("failed to parse [KernelCommandLine]: %w", err)
	}

	// Now validate the resulting unmarshaled object
	err = k.IsValid()
	if err != nil {
		return fmt.Errorf("failed to parse [KernelCommandLine]: %w", err)
	}
	return
}

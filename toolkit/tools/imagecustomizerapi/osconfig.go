// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"fmt"
	"strings"

	"github.com/asaskevich/govalidator"
)

// OSConfig defines how each system present on the image is supposed to be configured.
type OSConfig struct {
	Hostname             string             `yaml:"hostname"`
	Packages             Packages           `yaml:"packages"`
	SELinux              SELinux            `yaml:"selinux"`
	KernelCommandLine    KernelCommandLine  `yaml:"kernelCommandLine"`
	AdditionalFiles      AdditionalFilesMap `yaml:"additionalFiles"`
	PostInstallScripts   []Script           `yaml:"postInstallScripts"`
	FinalizeImageScripts []Script           `yaml:"finalizeImageScripts"`
	Users                []User             `yaml:"users"`
	Services             Services           `yaml:"services"`
	Modules              Modules            `yaml:"modules"`
	Verity               *Verity            `yaml:"verity"`
}

func (s *OSConfig) IsValid() error {
	var err error

	if s.Hostname != "" {
		if !govalidator.IsDNSName(s.Hostname) || strings.Contains(s.Hostname, "_") {
			return fmt.Errorf("invalid hostname: %s", s.Hostname)
		}
	}

	err = s.SELinux.IsValid()
	if err != nil {
		return fmt.Errorf("invalid selinux: %w", err)
	}

	err = s.KernelCommandLine.IsValid()
	if err != nil {
		return fmt.Errorf("invalid kernelCommandLine: %w", err)
	}

	err = s.AdditionalFiles.IsValid()
	if err != nil {
		return fmt.Errorf("invalid AdditionalFiles: %w", err)
	}

	for i, script := range s.PostInstallScripts {
		err = script.IsValid()
		if err != nil {
			return fmt.Errorf("invalid postInstallScripts item at index %d: %w", i, err)
		}
	}

	for i, script := range s.FinalizeImageScripts {
		err = script.IsValid()
		if err != nil {
			return fmt.Errorf("invalid finalizeImageScripts item at index %d: %w", i, err)
		}
	}

	for i, user := range s.Users {
		err = user.IsValid()
		if err != nil {
			return fmt.Errorf("invalid users item at index %d: %w", i, err)
		}
	}

	if err := s.Services.IsValid(); err != nil {
		return err
	}

	if err := s.Modules.IsValid(); err != nil {
		return err
	}

	if s.Verity != nil {
		err = s.Verity.IsValid()
		if err != nil {
			return fmt.Errorf("invalid verity: %w", err)
		}
	}

	return nil
}

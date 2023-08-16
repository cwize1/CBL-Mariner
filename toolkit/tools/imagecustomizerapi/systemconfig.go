// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"fmt"
	"strings"

	"github.com/asaskevich/govalidator"
)

// SystemConfig defines how each system present on the image is supposed to be configured.
type SystemConfig struct {
	Hostname          string                    `yaml:"Hostname"`
	KernelCommandLine KernelCommandLine         `yaml:"KernelCommandLine"`
	AdditionalFiles   map[string]FileConfigList `yaml:"AdditionalFiles"`
}

func (s *SystemConfig) IsValid() error {
	var err error

	if s.Hostname != "" {
		if !govalidator.IsDNSName(s.Hostname) || strings.Contains(s.Hostname, "_") {
			return fmt.Errorf("invalid hostname: %s", s.Hostname)
		}
	}

	err = s.KernelCommandLine.IsValid()
	if err != nil {
		return fmt.Errorf("invalid KernelCommandLine: %w", err)
	}

	for sourcePath, fileConfigList := range s.AdditionalFiles {
		err = fileConfigList.IsValid()
		if err != nil {
			return fmt.Errorf("invalid file configs for (%s): %w", sourcePath, err)
		}
	}

	return nil
}

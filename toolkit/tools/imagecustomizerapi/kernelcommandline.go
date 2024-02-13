// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"fmt"
	"strings"
)

type KernelCommandLine struct {
	// SELinux specifies whether or not to enable SELinux on the image (and what mode SELinux should be in).
	SELinuxMode SELinuxMode `yaml:"seLinuxMode"`
	// Extra kernel command line args.
	ExtraCommandLine string `yaml:"extraCommandLine"`
}

func (s *KernelCommandLine) IsValid() error {
	err := s.SELinuxMode.IsValid()
	if err != nil {
		return err
	}

	err = commandLineIsValid(s.ExtraCommandLine, "extraCommandLine")
	if err != nil {
		return err
	}

	return nil
}

func commandLineIsValid(commandLine string, fieldName string) error {
	// Disallow special characters to avoid breaking the grub.cfg file.
	// In addition, disallow the "`" character, since it is used as the sed escape character by
	// `installutils.setGrubCfgAdditionalCmdLine()`.
	if strings.ContainsAny(commandLine, "\n'\"\\$`") {
		return fmt.Errorf("the %s value contains invalid characters", fieldName)
	}

	return nil
}

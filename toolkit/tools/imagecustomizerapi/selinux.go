// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

type SELinux struct {
	// Mode specifies whether or not to enable SELinux on the image (and what mode SELinux should be in).
	Mode SELinuxMode `yaml:"mode"`
}

func (s *SELinux) IsValid() error {
	err := s.Mode.IsValid()
	if err != nil {
		return err
	}

	return nil
}

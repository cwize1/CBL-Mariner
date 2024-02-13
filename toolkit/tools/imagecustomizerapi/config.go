// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"fmt"
)

type Config struct {
	Storage  *Storage `yaml:"storage"`
	Iso      *Iso     `yaml:"iso"`
	OSConfig OSConfig `yaml:"osConfig"`
}

func (c *Config) IsValid() (err error) {
	if c.Storage != nil {
		err = c.Storage.IsValid()
		if err != nil {
			return fmt.Errorf("invalid storage value:\n%w", err)
		}
	}

	if c.Iso != nil {
		err = c.Iso.IsValid()
		if err != nil {
			return err
		}
	}

	err = c.OSConfig.IsValid()
	if err != nil {
		return err
	}

	return nil
}

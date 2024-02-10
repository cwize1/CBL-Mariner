// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"fmt"
)

type Config struct {
	Storage      *Storage     `yaml:"storage"`
	Iso          *Iso         `yaml:"iso"`
	SystemConfig SystemConfig `yaml:"systemConfig"`
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

	err = c.SystemConfig.IsValid()
	if err != nil {
		return err
	}

	return nil
}

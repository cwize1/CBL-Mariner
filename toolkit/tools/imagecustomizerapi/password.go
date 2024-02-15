// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"fmt"
)

type Password struct {
	Type  PasswordType `yaml:"type"`
	Value string       `yaml:"value"`
}

func (p *Password) IsValid() error {
	err := p.Type.IsValid()
	if err != nil {
		return fmt.Errorf("invalid type value:\n%w", err)
	}

	switch p.Type {
	case PasswordTypeDefault, PasswordTypeLocked:
		if p.Value != "" {
			return fmt.Errorf("value must be empty when type is (%s)", p.Value)
		}

	case PasswordTypePlainText, PasswordTypeHashed, PasswordTypePlainTextFile, PasswordTypeHashedFile:
		if p.Value == "" {
			return fmt.Errorf("value must not be empty when type is (%s)", p.Value)
		}
	}

	return nil
}

// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"fmt"
)

type PasswordType string

const (
	PasswordTypeDefault       PasswordType = ""
	PasswordTypeLocked        PasswordType = "locked"
	PasswordTypePlainText     PasswordType = "plain-text"
	PasswordTypeHashed        PasswordType = "hashed"
	PasswordTypePlainTextFile PasswordType = "plain-text-file"
	PasswordTypeHashedFile    PasswordType = "hashed-file"
)

func (t PasswordType) IsValid() error {
	switch t {
	case PasswordTypeDefault, PasswordTypeLocked, PasswordTypePlainText, PasswordTypeHashed, PasswordTypePlainTextFile,
		PasswordTypeHashedFile:
		// All good.
		return nil

	default:
		return fmt.Errorf("invalid password type value (%v)", t)
	}
}

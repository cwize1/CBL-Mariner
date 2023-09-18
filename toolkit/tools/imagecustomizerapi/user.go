// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

type User struct {
	Name                string   `json:"Name"`
	UID                 string   `json:"UID"`
	PasswordHashed      bool     `json:"PasswordHashed"`
	PasswordPath        string   `json:"PasswordPath"`
	PasswordExpiresDays int64    `json:"PasswordExpiresDays"`
	SSHPubKeyPaths      []string `json:"SSHPubKeyPaths"`
	PrimaryGroup        string   `json:"PrimaryGroup"`
	SecondaryGroups     []string `json:"SecondaryGroups"`
	StartupCommand      string   `json:"StartupCommand"`
}

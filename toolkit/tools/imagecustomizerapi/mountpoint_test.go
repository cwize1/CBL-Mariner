// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMountPointIsValidInvalidMountIdentifier(t *testing.T) {
	mountPoint := MountPoint{
		DeviceId:            "a",
		FileSystemType:      "fat32",
		MountIdentifierType: "bad",
	}

	err := mountPoint.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid")
	assert.ErrorContains(t, err, "mountIdentifierType")
}

func TestMountPointIsValidUnsupportedFileSystem(t *testing.T) {
	mountPoint := MountPoint{
		DeviceId:       "a",
		FileSystemType: "bad",
	}

	err := mountPoint.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid")
	assert.ErrorContains(t, err, "fileSystemType")
}

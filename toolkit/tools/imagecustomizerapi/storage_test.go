// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemConfigIsValidDuplicatePartitionID(t *testing.T) {
	value := Storage{
		Disks: []Disk{
			{
				PartitionTableType: PartitionTableTypeGpt,
				MaxSize:            2,
				Partitions: []Partition{
					{
						ID:    "a",
						Start: 1,
					},
				},
			},
		},
		BootType: "efi",
		MountPoints: []MountPoint{
			{
				ID:             "a",
				FilesystemType: "ext4",
			},
			{
				ID:             "a",
				FilesystemType: "ext4",
			},
		},
	}

	err := value.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "duplicate mountPoints ID")
}

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
						ID:             "a",
						FileSystemType: "ext4",
						Start:          1,
					},
				},
			},
		},
		BootType: "efi",
		MountPoints: []MountPoint{
			{
				DeviceId: "a",
			},
			{
				DeviceId: "a",
			},
		},
	}

	err := value.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "duplicate mountPoints ID")
}

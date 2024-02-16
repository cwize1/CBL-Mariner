// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorageIsValidDuplicatePartitionID(t *testing.T) {
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
		FileSystems: []FileSystem{
			{
				DeviceId:       "a",
				FileSystemType: "ext4",
			},
			{
				DeviceId:       "a",
				FileSystemType: "ext4",
			},
		},
	}

	err := value.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "duplicate mountPoints deviceId")
}

func TestStorageIsValidBadEspFsType(t *testing.T) {
	value := Storage{
		Disks: []Disk{
			{
				PartitionTableType: PartitionTableTypeGpt,
				MaxSize:            2,
				Partitions: []Partition{
					{
						ID:                "a",
						Start:             1,
						BootPartitionType: "esp",
					},
				},
			},
		},
		BootType: "efi",
		FileSystems: []FileSystem{
			{
				DeviceId:       "a",
				FileSystemType: "ext4",
			},
		},
	}

	err := value.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "ESP")
	assert.ErrorContains(t, err, "fat32")
}

func TestStorageIsValidBadBiosBootFsType(t *testing.T) {
	value := Storage{
		Disks: []Disk{
			{
				PartitionTableType: PartitionTableTypeGpt,
				MaxSize:            2,
				Partitions: []Partition{
					{
						ID:                "a",
						Start:             1,
						BootPartitionType: "bios-grub",
					},
				},
			},
		},
		BootType: "legacy",
		FileSystems: []FileSystem{
			{
				DeviceId:       "a",
				FileSystemType: "ext4",
			},
		},
	}

	err := value.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "BIOS boot")
	assert.ErrorContains(t, err, "fat32")
}

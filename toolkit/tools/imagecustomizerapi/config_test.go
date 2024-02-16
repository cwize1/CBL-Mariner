// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigIsValid(t *testing.T) {
	config := &Config{
		Storage: &Storage{
			Disks: []Disk{{
				PartitionTableType: "gpt",
				MaxSize:            2,
				Partitions: []Partition{
					{
						ID:                "esp",
						Start:             1,
						BootPartitionType: "esp",
					},
				},
			}},
			BootType: "efi",
			FileSystems: []FileSystem{
				{
					DeviceId:       "esp",
					Path:           "/boot/efi",
					FileSystemType: "fat32",
				},
			},
		},
		OS: OS{
			Hostname: "test",
		},
	}

	err := config.IsValid()
	assert.NoError(t, err)
}

func TestConfigIsValidLegacy(t *testing.T) {
	config := &Config{
		Storage: &Storage{
			Disks: []Disk{{
				PartitionTableType: "gpt",
				MaxSize:            2,
				Partitions: []Partition{
					{
						ID:                "boot",
						Start:             1,
						BootPartitionType: "bios-grub",
					},
				},
			}},
			BootType: "legacy",
			FileSystems: []FileSystem{
				{
					DeviceId:       "boot",
					FileSystemType: "fat32",
				},
			},
		},
		OS: OS{
			Hostname: "test",
		},
	}

	err := config.IsValid()
	assert.NoError(t, err)
}

func TestConfigIsValidNoBootType(t *testing.T) {
	config := &Config{
		Storage: &Storage{
			Disks: []Disk{{
				PartitionTableType: "gpt",
				MaxSize:            2,
				Partitions: []Partition{
					{
						ID:    "a",
						Start: 1,
					},
				},
			}},
			FileSystems: []FileSystem{
				{
					DeviceId:       "a",
					FileSystemType: "ext4",
				},
			},
		},
		OS: OS{
			Hostname: "test",
		},
	}

	err := config.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "bootType")
}

func TestConfigIsValidMultipleDisks(t *testing.T) {
	config := &Config{
		Storage: &Storage{
			Disks: []Disk{
				{
					PartitionTableType: "gpt",
					MaxSize:            1,
				},
				{
					PartitionTableType: "gpt",
					MaxSize:            1,
				},
			},
		},
		OS: OS{
			Hostname: "test",
		},
	}

	err := config.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "multiple disks")
}

func TestConfigIsValidZeroDisks(t *testing.T) {
	config := &Config{
		Storage: &Storage{
			Disks: []Disk{},
		},
		OS: OS{
			Hostname: "test",
		},
	}

	err := config.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "1 disk")
}

func TestConfigIsValidBadHostname(t *testing.T) {
	config := &Config{
		OS: OS{
			Hostname: "test_",
		},
	}

	err := config.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid hostname")
}

func TestConfigIsValidBadDisk(t *testing.T) {
	config := &Config{
		Storage: &Storage{
			Disks: []Disk{{
				PartitionTableType: PartitionTableTypeGpt,
				MaxSize:            0,
			}},
		},
		OS: OS{
			Hostname: "test",
		},
	}

	err := config.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "MaxSize")
}

func TestConfigIsValidMissingEsp(t *testing.T) {
	config := &Config{
		Storage: &Storage{

			Disks: []Disk{{
				PartitionTableType: "gpt",
				MaxSize:            2,
				Partitions:         []Partition{},
			}},
			BootType: "efi",
		},
		OS: OS{
			Hostname: "test",
		},
	}

	err := config.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "esp")
	assert.ErrorContains(t, err, "efi")
}

func TestConfigIsValidMissingBiosBoot(t *testing.T) {
	config := &Config{
		Storage: &Storage{

			Disks: []Disk{{
				PartitionTableType: "gpt",
				MaxSize:            2,
				Partitions:         []Partition{},
			}},
			BootType: "legacy",
		},
		OS: OS{
			Hostname: "test",
		},
	}

	err := config.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "bios_grub")
	assert.ErrorContains(t, err, "legacy")
}

func TestConfigIsValidInvalidMountPoint(t *testing.T) {
	config := &Config{
		Storage: &Storage{
			Disks: []Disk{{
				PartitionTableType: "gpt",
				MaxSize:            2,
				Partitions: []Partition{
					{
						ID:                "esp",
						Start:             1,
						BootPartitionType: BootPartitionTypeESP,
					},
				},
			}},
			BootType: "efi",
			FileSystems: []FileSystem{
				{
					DeviceId:       "esp",
					FileSystemType: "fat32",
					Path:           "boot/efi",
				},
			},
		},
		OS: OS{
			Hostname: "test",
		},
	}

	err := config.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "absolute path")
}

func TestConfigIsValidInvalidPartitionId(t *testing.T) {
	config := &Config{
		Storage: &Storage{
			Disks: []Disk{{
				PartitionTableType: "gpt",
				MaxSize:            2,
				Partitions: []Partition{
					{
						ID:                "esp",
						Start:             1,
						BootPartitionType: BootPartitionTypeESP,
					},
				},
			}},
			BootType: "efi",
			FileSystems: []FileSystem{
				{
					DeviceId:       "boot",
					FileSystemType: "fat32",
					Path:           "/boot/efi",
				},
			},
		},
		OS: OS{
			Hostname: "test",
		},
	}

	err := config.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "partition")
	assert.ErrorContains(t, err, "ID")
}

func TestConfigIsValidKernelCLI(t *testing.T) {
	config := &Config{
		Storage: &Storage{
			Disks: []Disk{{
				PartitionTableType: "gpt",
				MaxSize:            2,
				Partitions: []Partition{
					{
						ID:                "esp",
						Start:             1,
						BootPartitionType: BootPartitionTypeESP,
					},
				},
			}},
			BootType: "efi",
			FileSystems: []FileSystem{
				{
					DeviceId:       "esp",
					FileSystemType: "fat32",
					Path:           "/boot/efi",
				},
			},
		},
		OS: OS{
			Hostname: "test",
			KernelCommandLine: KernelCommandLine{
				ExtraCommandLine: "console=ttyS0",
			},
		},
	}
	err := config.IsValid()
	assert.NoError(t, err)
}

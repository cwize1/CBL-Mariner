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
						ID:     "esp",
						FsType: "fat32",
						Start:  1,
						Flags: []PartitionFlag{
							"esp",
							"boot",
						},
					},
				},
			}},
			BootType: "efi",
			MountPoints: []MountPoint{
				{
					ID:   "esp",
					Path: "/boot/efi",
				},
			},
		},
		SystemConfig: SystemConfig{
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
						ID:     "boot",
						FsType: "fat32",
						Start:  1,
						Flags: []PartitionFlag{
							"bios_grub",
						},
					},
				},
			}},
			BootType: "legacy",
		},
		SystemConfig: SystemConfig{
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
						ID:     "a",
						FsType: "ext4",
						Start:  1,
					},
				},
			}},
			MountPoints: []MountPoint{
				{
					ID: "a",
				},
			},
		},
		SystemConfig: SystemConfig{
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
		SystemConfig: SystemConfig{
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
		SystemConfig: SystemConfig{
			Hostname: "test",
		},
	}

	err := config.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "1 disk")
}

func TestConfigIsValidBadHostname(t *testing.T) {
	config := &Config{
		SystemConfig: SystemConfig{
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
		SystemConfig: SystemConfig{
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
		SystemConfig: SystemConfig{
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
		SystemConfig: SystemConfig{
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
						ID:     "esp",
						FsType: "fat32",
						Start:  1,
						Flags: []PartitionFlag{
							"esp",
							"boot",
						},
					},
				},
			}},
			BootType: "efi",
			MountPoints: []MountPoint{
				{
					ID:   "esp",
					Path: "boot/efi",
				},
			},
		},
		SystemConfig: SystemConfig{
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
						ID:     "esp",
						FsType: "fat32",
						Start:  1,
						Flags: []PartitionFlag{
							"esp",
							"boot",
						},
					},
				},
			}},
			BootType: "efi",
			MountPoints: []MountPoint{
				{
					ID:   "boot",
					Path: "/boot/efi",
				},
			},
		},
		SystemConfig: SystemConfig{
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
						ID:     "esp",
						FsType: "fat32",
						Start:  1,
						Flags: []PartitionFlag{
							"esp",
							"boot",
						},
					},
				},
			}},
			BootType: "efi",
			MountPoints: []MountPoint{
				{
					ID:   "esp",
					Path: "/boot/efi",
				},
			},
		},
		SystemConfig: SystemConfig{
			Hostname: "test",
			KernelCommandLine: KernelCommandLine{
				ExtraCommandLine: "console=ttyS0",
			},
		},
	}
	err := config.IsValid()
	assert.NoError(t, err)
}

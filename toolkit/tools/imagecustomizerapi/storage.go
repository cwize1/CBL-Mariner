// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"fmt"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/sliceutils"
)

type Storage struct {
	BootType    BootType     `yaml:"bootType"`
	Disks       []Disk       `yaml:"disks"`
	MountPoints []MountPoint `yaml:"mountPoints"`
}

func (s *Storage) IsValid() (err error) {
	disks := s.Disks
	if len(disks) < 1 {
		return fmt.Errorf("at least 1 disk must be specified (or the storage field should be ommited)")
	}
	if len(disks) > 1 {
		return fmt.Errorf("multiple disks is not currently supported")
	}

	// Verify disks are valid.
	for i, disk := range disks {
		err := disk.IsValid()
		if err != nil {
			return fmt.Errorf("invalid disk at index %d:\n%w", i, err)
		}
	}

	// Verify boot type is valid.
	err = s.BootType.IsValid()
	if err != nil {
		return err
	}

	partitionSet := make(map[string]Partition)
	for _, disk := range disks {
		for _, partition := range disk.Partitions {
			if _, existingName := partitionSet[partition.ID]; existingName {
				return fmt.Errorf("duplicate partition ID (%s)", partition.ID)
			}

			partitionSet[partition.ID] = partition
		}
	}

	// Verify the mount point settings are valid.
	mountPointIDSet := make(map[string]MountPoint)
	for i, mountPoint := range s.MountPoints {
		err = mountPoint.IsValid()
		if err != nil {
			return fmt.Errorf("invalid mountPoints item at index %d: %w", i, err)
		}

		if _, existingName := mountPointIDSet[mountPoint.DeviceId]; existingName {
			return fmt.Errorf("duplicate mountPoints deviceId used (%s) at index %d", mountPoint.DeviceId, i)
		}

		mountPointIDSet[mountPoint.DeviceId] = mountPoint

		// Ensure there is a partition with the same ID.
		_, foundPartition := partitionSet[mountPoint.DeviceId]
		if !foundPartition {
			return fmt.Errorf("invalid mount point at index %d:\nno partition with matching ID (%s)", i,
				mountPoint.DeviceId)
		}
	}

	// Ensure special partitions have the correct filesystem type.
	for _, disk := range disks {
		for _, partition := range disk.Partitions {
			mountPoint, hasMountPoint := mountPointIDSet[partition.ID]

			if partition.IsESP() {
				if !hasMountPoint || mountPoint.FileSystemType != FileSystemTypeFat32 {
					return fmt.Errorf("ESP partition must have 'fat32' filesystem type")
				}
			}

			if partition.IsBiosBoot() {
				if !hasMountPoint || mountPoint.FileSystemType != FileSystemTypeFat32 {
					return fmt.Errorf("BIOS boot partition must have 'fat32' filesystem type")
				}
			}
		}
	}

	// Ensure the correct partitions exist to support the specified the boot type.
	switch s.BootType {
	case BootTypeEfi:
		hasEsp := sliceutils.ContainsFunc(s.Disks, func(disk Disk) bool {
			return sliceutils.ContainsFunc(disk.Partitions, func(partition Partition) bool {
				return partition.IsESP()
			})
		})
		if !hasEsp {
			return fmt.Errorf("'esp' partition must be provided for 'efi' boot type")
		}

	case BootTypeLegacy:
		hasBiosBoot := sliceutils.ContainsFunc(s.Disks, func(disk Disk) bool {
			return sliceutils.ContainsFunc(disk.Partitions, func(partition Partition) bool {
				return partition.IsBiosBoot()
			})
		})
		if !hasBiosBoot {
			return fmt.Errorf("'bios_grub' partition must be provided for 'legacy' boot type")
		}
	}

	return nil
}

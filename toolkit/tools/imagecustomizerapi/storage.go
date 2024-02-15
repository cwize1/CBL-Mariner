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

	// Verify the partition settings are valid.
	partitionIDSet := make(map[string]bool)
	for i, partition := range s.MountPoints {
		err = partition.IsValid()
		if err != nil {
			return fmt.Errorf("invalid mountPoints item at index %d: %w", i, err)
		}

		if _, existingName := partitionIDSet[partition.DeviceId]; existingName {
			return fmt.Errorf("duplicate mountPoints deviceId used (%s) at index %d", partition.DeviceId, i)
		}

		partitionIDSet[partition.DeviceId] = false // dummy value
	}

	// Ensure all the partition settings object have an equivalent partition object.
	for i, mountPoint := range s.MountPoints {
		diskExists := sliceutils.ContainsFunc(s.Disks, func(disk Disk) bool {
			return sliceutils.ContainsFunc(disk.Partitions, func(partition Partition) bool {
				return partition.ID == mountPoint.DeviceId
			})
		})
		if !diskExists {
			return fmt.Errorf("invalid mount point at index %d:\nno partition with matching ID (%s)", i,
				mountPoint.DeviceId)
		}
	}

	// Ensure the correct partitions exist to support the specified the boot type.
	switch s.BootType {
	case BootTypeEfi:
		hasEsp := sliceutils.ContainsFunc(s.Disks, func(disk Disk) bool {
			return sliceutils.ContainsFunc(disk.Partitions, func(partition Partition) bool {
				return sliceutils.ContainsValue(partition.Flags, PartitionFlagESP)
			})
		})
		if !hasEsp {
			return fmt.Errorf("'esp' partition must be provided for 'efi' boot type")
		}

	case BootTypeLegacy:
		hasBiosBoot := sliceutils.ContainsFunc(s.Disks, func(disk Disk) bool {
			return sliceutils.ContainsFunc(disk.Partitions, func(partition Partition) bool {
				return sliceutils.ContainsValue(partition.Flags, PartitionFlagBiosGrub)
			})
		})
		if !hasBiosBoot {
			return fmt.Errorf("'bios_grub' partition must be provided for 'legacy' boot type")
		}
	}

	return nil
}

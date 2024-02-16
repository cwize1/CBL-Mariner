// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerlib

import (
	"fmt"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/imagecustomizerapi"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/imagegen/configuration"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/sliceutils"
)

func bootTypeToImager(bootType imagecustomizerapi.BootType) (string, error) {
	switch bootType {
	case imagecustomizerapi.BootTypeEfi:
		return "efi", nil

	case imagecustomizerapi.BootTypeLegacy:
		return "legacy", nil

	default:
		return "", fmt.Errorf("invalid bootType value (%s)", bootType)
	}
}

func diskConfigToImager(diskConfig imagecustomizerapi.Disk, fileSystems []imagecustomizerapi.FileSystem,
) (configuration.Disk, error) {
	imagerPartitionTableType, err := partitionTableTypeToImager(diskConfig.PartitionTableType)
	if err != nil {
		return configuration.Disk{}, err
	}

	imagerPartitions, err := partitionsToImager(diskConfig.Partitions, fileSystems)
	if err != nil {
		return configuration.Disk{}, err
	}

	imagerDisk := configuration.Disk{
		PartitionTableType: imagerPartitionTableType,
		MaxSize:            diskConfig.MaxSize,
		Partitions:         imagerPartitions,
	}
	return imagerDisk, err
}

func partitionTableTypeToImager(partitionTableType imagecustomizerapi.PartitionTableType,
) (configuration.PartitionTableType, error) {
	switch partitionTableType {
	case imagecustomizerapi.PartitionTableTypeGpt:
		return configuration.PartitionTableTypeGpt, nil

	default:
		return "", fmt.Errorf("unknown partition table type (%s)", partitionTableType)
	}
}

func partitionsToImager(partitions []imagecustomizerapi.Partition, fileSystems []imagecustomizerapi.FileSystem) ([]configuration.Partition, error) {
	imagerPartitions := []configuration.Partition(nil)
	for _, partition := range partitions {
		imagerPartition, err := partitionToImager(partition, fileSystems)
		if err != nil {
			return nil, err
		}

		imagerPartitions = append(imagerPartitions, imagerPartition)
	}

	return imagerPartitions, nil
}

func partitionToImager(partition imagecustomizerapi.Partition, fileSystems []imagecustomizerapi.FileSystem,
) (configuration.Partition, error) {
	fileSystem, foundMountPoint := sliceutils.FindValueFunc(fileSystems, func(fileSystem imagecustomizerapi.FileSystem) bool {
		return fileSystem.DeviceId == partition.ID
	})
	if !foundMountPoint {
		return configuration.Partition{}, fmt.Errorf("failed to find mount point with ID (%s)", partition.ID)
	}

	imagerEnd, _ := partition.GetEnd()

	imagerFlags, err := partitionFlagsToImager(partition.Flags)
	if err != nil {
		return configuration.Partition{}, err
	}

	imagerPartition := configuration.Partition{
		ID:     partition.ID,
		FsType: string(fileSystem.FileSystemType),
		Name:   partition.Label,
		Start:  partition.Start,
		End:    imagerEnd,
		Flags:  imagerFlags,
	}
	return imagerPartition, nil
}

func partitionFlagsToImager(flags []imagecustomizerapi.PartitionFlag) ([]configuration.PartitionFlag, error) {
	imagerFlags := []configuration.PartitionFlag(nil)
	for _, flag := range flags {
		imagerFlag, err := partitionFlagToImager(flag)
		if err != nil {
			return nil, err
		}

		imagerFlags = append(imagerFlags, imagerFlag)
	}

	return imagerFlags, nil
}

func partitionFlagToImager(flag imagecustomizerapi.PartitionFlag) (configuration.PartitionFlag, error) {
	switch flag {
	case imagecustomizerapi.PartitionFlagESP:
		return configuration.PartitionFlagESP, nil

	case imagecustomizerapi.PartitionFlagBiosGrub:
		return configuration.PartitionFlagBiosGrub, nil

	case imagecustomizerapi.PartitionFlagBoot:
		return configuration.PartitionFlagBoot, nil

	default:
		return "", fmt.Errorf("unknown partition flag (%s)", flag)
	}
}

func fileSystemsToImager(fileSystems []imagecustomizerapi.FileSystem,
) ([]configuration.PartitionSetting, error) {
	imagerPartitionSettings := []configuration.PartitionSetting(nil)
	for _, mountPoint := range fileSystems {
		imagerPartitionSetting, err := fileSystemToImager(mountPoint)
		if err != nil {
			return nil, err
		}
		imagerPartitionSettings = append(imagerPartitionSettings, imagerPartitionSetting)
	}
	return imagerPartitionSettings, nil
}

func fileSystemToImager(mountPoint imagecustomizerapi.FileSystem,
) (configuration.PartitionSetting, error) {
	imagerMountIdentifierType, err := mountIdentifierTypeToImager(mountPoint.MountIdentifierType)
	if err != nil {
		return configuration.PartitionSetting{}, err
	}

	imagerPartitionSetting := configuration.PartitionSetting{
		ID:              mountPoint.DeviceId,
		MountIdentifier: imagerMountIdentifierType,
		MountOptions:    mountPoint.Options,
		MountPoint:      mountPoint.Path,
	}
	return imagerPartitionSetting, nil
}

func mountIdentifierTypeToImager(mountIdentifierType imagecustomizerapi.MountIdentifierType,
) (configuration.MountIdentifier, error) {
	switch mountIdentifierType {
	case imagecustomizerapi.MountIdentifierTypeUuid:
		return configuration.MountIdentifierUuid, nil

	case imagecustomizerapi.MountIdentifierTypePartUuid, imagecustomizerapi.MountIdentifierTypeDefault:
		return configuration.MountIdentifierPartUuid, nil

	case imagecustomizerapi.MountIdentifierTypePartLabel:
		return configuration.MountIdentifierPartLabel, nil

	default:
		return "", fmt.Errorf("unknown MountIdentifierType value (%s)", mountIdentifierType)
	}
}

func kernelCommandLineToImager(kernelCommandLine imagecustomizerapi.KernelCommandLine,
	selinux imagecustomizerapi.SELinux, currentSELinuxMode imagecustomizerapi.SELinuxMode,
) (configuration.KernelCommandLine, error) {
	imagerSELinux, err := selinuxModeMaybeDefaultToImager(selinux.Mode, currentSELinuxMode)
	if err != nil {
		return configuration.KernelCommandLine{}, err
	}

	imagerKernelCommandLine := configuration.KernelCommandLine{
		ExtraCommandLine: kernelCommandLine.ExtraCommandLine,
		SELinux:          imagerSELinux,
		SELinuxPolicy:    "",
	}
	return imagerKernelCommandLine, nil
}

func selinuxModeMaybeDefaultToImager(selinuxMode imagecustomizerapi.SELinuxMode,
	currentSELinuxMode imagecustomizerapi.SELinuxMode,
) (configuration.SELinux, error) {
	if selinuxMode == imagecustomizerapi.SELinuxModeDefault {
		selinuxMode = currentSELinuxMode
	}

	return selinuxModeToImager(selinuxMode)
}

func selinuxModeToImager(selinuxMode imagecustomizerapi.SELinuxMode) (configuration.SELinux, error) {
	switch selinuxMode {
	case imagecustomizerapi.SELinuxModeDisabled:
		return configuration.SELinuxOff, nil

	case imagecustomizerapi.SELinuxModeEnforcing:
		return configuration.SELinuxEnforcing, nil

	case imagecustomizerapi.SELinuxModePermissive:
		return configuration.SELinuxPermissive, nil

	case imagecustomizerapi.SELinuxModeForceEnforcing:
		return configuration.SELinuxForceEnforcing, nil

	default:
		return "", fmt.Errorf("unknown SELinux value (%s)", selinuxMode)
	}
}

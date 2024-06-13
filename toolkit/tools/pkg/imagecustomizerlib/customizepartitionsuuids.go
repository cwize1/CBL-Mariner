// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerlib

import (
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/microsoft/azurelinux/toolkit/tools/imagecustomizerapi"
	"github.com/microsoft/azurelinux/toolkit/tools/imagegen/diskutils"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/logger"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/shell"
)

func handleResetPartitionsUuids(resetPartitionsUuidsType imagecustomizerapi.ResetPartitionsUuidsType,
	imageConnection *ImageConnection,
) error {
	if resetPartitionsUuidsType == imagecustomizerapi.ResetPartitionsUuidsTypeDefault {
		return nil
	}

	logger.Log.Infof("Resetting partition UUIDs")

	partitions, err := diskutils.GetDiskPartitions(imageConnection.Loopback().DevicePath())
	if err != nil {
		return err
	}

	for _, partition := range partitions {
		if partition.Type != "part" {
			continue
		}

		err = resetFileSystemUuid(partition)
		if err != nil {
			return fmt.Errorf("failed to reset partition's (%s) filesystem (%s) UUID:\n%w", partition.Path,
				partition.FileSystemType, err)
		}
	}

	for partNum, partition := range partitions {
		if partition.Type != "part" {
			continue
		}

		err = resetPartitionUuid(imageConnection.Loopback().DevicePath(), partNum)
		if err != nil {
			return fmt.Errorf("failed to update partition (%s) UUID:\n%w", partition.Path, err)
		}
	}

	return nil
}

func resetFileSystemUuid(partition diskutils.PartitionInfo) error {
	switch partition.FileSystemType {
	case "ext2", "ext3", "ext4":
		// tune2fs requires you to run 'e2fsck' first.
		err := shell.ExecuteLive(true /*squashErrors*/, "e2fsck", "-fy", partition.Path)
		if err != nil {
			return fmt.Errorf("failed to check %s with e2fsck:\n%w", partition.Path, err)
		}

		err = shell.ExecuteLive(true /*squashErrors*/, "tune2fs", "-U", uuid.New().String(), partition.Path)
		if err != nil {
			return err
		}

	case "xfs":
		err := shell.ExecuteLive(true /*squashErrors*/, "xfs_admin", "-U", uuid.New().String(), partition.Path)
		if err != nil {
			return err
		}

	case "vfat":
		err := shell.ExecuteLive(true /*squashErrors*/, "fatlabel", "-i", "-r", partition.Path)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("unsupported filesystem type (%s)", partition.FileSystemType)
	}

	return nil
}

func resetPartitionUuid(device string, partNum int) error {
	err := shell.ExecuteLive(true /*squashErrors*/, "sfdisk", "--part-uuid", device, strconv.Itoa(partNum),
		uuid.New().String())
	if err != nil {
		return err
	}

	return nil
}

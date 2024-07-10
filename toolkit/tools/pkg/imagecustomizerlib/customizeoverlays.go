// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerlib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/microsoft/azurelinux/toolkit/tools/imagecustomizerapi"
	"github.com/microsoft/azurelinux/toolkit/tools/imagegen/diskutils"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/safechroot"
)

func handleOverlays(overlays []imagecustomizerapi.Overlay, imageChroot *safechroot.Chroot) error {
	if len(overlays) <= 0 {
		return nil
	}

	fstabFile := filepath.Join(imageChroot.RootDir(), "/etc/fstab")
	fstabEntries, err := diskutils.ReadFstabFile(fstabFile)
	if err != nil {
		return fmt.Errorf("failed to read fstab file:\n%w", err)
	}

	for _, overlay := range overlays {
		options := overlay.Options
		if options != "" {
			options += ","
		}
		options += fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", overlay.Lower, overlay.Upper, overlay.Work)

		newEntry := diskutils.FstabEntry{
			Source:  "overlay",
			Target:  overlay.Target,
			FsType:  "overlay",
			Options: options,
			Freq:    0,
			PassNo:  2,
		}

		fstabEntries = append(fstabEntries, newEntry)

		err := imageChroot.UnsafeRun(func() error {
			return os.MkdirAll(overlay.Upper, 0o755)
		})
		if err != nil {
			return fmt.Errorf("failed to create overlay upper directory:\n%w", err)
		}

		err = imageChroot.UnsafeRun(func() error {
			return os.MkdirAll(overlay.Work, 0o755)
		})
		if err != nil {
			return fmt.Errorf("failed to create overlay work directory:\n%w", err)
		}
	}

	// Write the updated fstab entries back to the fstab file
	err = diskutils.WriteFstabFile(fstabEntries, fstabFile)
	if err != nil {
		return err
	}

	return nil
}

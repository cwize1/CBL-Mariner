// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

// Package that assists with mounting and unmounting cleanly.
package safemount

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

type Mount struct {
	target    string
	isMounted bool
}

func NewMount(source, target, fstype string, flags uintptr, data string) (*Mount, error) {
	var err error

	err = os.MkdirAll(target, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create mount directory (%s): %w", target, err)
	}

	err = unix.Mount(source, target, fstype, flags, data)
	if err != nil {
		return nil, fmt.Errorf("failed to mount (%s) to (%s): %w", source, target, err)
	}

	mountHandle := &Mount{
		target:    target,
		isMounted: true,
	}

	return mountHandle, nil
}

func (m *Mount) Close() error {
	var err error

	if !m.isMounted {
		return nil
	}

	err = unix.Unmount(m.target, 0)
	if err != nil {
		return fmt.Errorf("failed to unmount (%s): %w", m.target, err)
	}

	m.isMounted = false
	return nil
}

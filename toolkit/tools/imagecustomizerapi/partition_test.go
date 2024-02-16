// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"testing"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/ptrutils"
	"github.com/stretchr/testify/assert"
)

func TestPartitionIsValidExpanding(t *testing.T) {
	partition := Partition{
		ID:    "a",
		Start: 0,
	}

	err := partition.IsValid()
	assert.NoError(t, err)
}

func TestPartitionIsValidFixedSize(t *testing.T) {
	partition := Partition{
		ID:    "a",
		Start: 0,
		End:   ptrutils.PtrTo(uint64(1)),
	}

	err := partition.IsValid()
	assert.NoError(t, err)
}

func TestPartitionIsValidZeroSize(t *testing.T) {
	partition := Partition{
		ID:    "a",
		Start: 0,
		End:   ptrutils.PtrTo(uint64(0)),
	}

	err := partition.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "partition")
	assert.ErrorContains(t, err, "size")
}

func TestPartitionIsValidZeroSizeV2(t *testing.T) {
	partition := Partition{
		ID:    "a",
		Start: 0,
		Size:  ptrutils.PtrTo(uint64(0)),
	}

	err := partition.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "partition")
	assert.ErrorContains(t, err, "size")
}

func TestPartitionIsValidNegativeSize(t *testing.T) {
	partition := Partition{
		ID:    "a",
		Start: 2,
		End:   ptrutils.PtrTo(uint64(1)),
	}

	err := partition.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "partition")
	assert.ErrorContains(t, err, "size")
}

func TestPartitionIsValidBothEndAndSize(t *testing.T) {
	partition := Partition{
		ID:    "a",
		Start: 2,
		End:   ptrutils.PtrTo(uint64(3)),
		Size:  ptrutils.PtrTo(uint64(1)),
	}

	err := partition.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "End")
	assert.ErrorContains(t, err, "Size")
}

func TestPartitionIsValidGoodName(t *testing.T) {
	partition := Partition{
		ID:    "a",
		Start: 0,
		End:   nil,
		Label: "a",
	}

	err := partition.IsValid()
	assert.NoError(t, err)
}

func TestPartitionIsValidNameTooLong(t *testing.T) {
	partition := Partition{
		ID:    "a",
		Start: 0,
		End:   nil,
		Label: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}

	err := partition.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "name")
	assert.ErrorContains(t, err, "too long")
}

func TestPartitionIsValidNameNonASCII(t *testing.T) {
	partition := Partition{
		ID:    "a",
		Start: 0,
		End:   nil,
		Label: "❤️",
	}

	err := partition.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "name")
	assert.ErrorContains(t, err, "ASCII")
}

func TestPartitionIsValidGoodFlag(t *testing.T) {
	partition := Partition{
		ID:    "a",
		Start: 0,
		End:   nil,
		Flags: []PartitionFlag{"esp"},
	}

	err := partition.IsValid()
	assert.NoError(t, err)
}

func TestPartitionIsValidBadFlag(t *testing.T) {
	partition := Partition{
		ID:    "a",
		Start: 0,
		End:   nil,
		Flags: []PartitionFlag{"a"},
	}

	err := partition.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "PartitionFlag")
}

func TestPartitionIsValidBadBiosBootStart(t *testing.T) {
	partition := Partition{
		ID:    "a",
		Start: 2,
		End:   nil,
		Flags: []PartitionFlag{"bios_grub"},
	}

	err := partition.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "BIOS boot")
	assert.ErrorContains(t, err, "start")
}

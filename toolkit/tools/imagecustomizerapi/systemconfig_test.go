// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"testing"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/ptrutils"
	"github.com/stretchr/testify/assert"
)

func TestSystemConfigValidEmpty(t *testing.T) {
	testValidYamlValue[*SystemConfig](t, "{ }", &SystemConfig{})
}

func TestSystemConfigValidHostname(t *testing.T) {
	testValidYamlValue[*SystemConfig](t, "{ \"Hostname\": \"validhostname\" }", &SystemConfig{Hostname: "validhostname"})
}

func TestSystemConfigInvalidHostname(t *testing.T) {
	testInvalidYamlValue[*SystemConfig](t, "{ \"Hostname\": \"invalid_hostname\" }")
}

func TestSystemConfigInvalidAdditionalFiles(t *testing.T) {
	testInvalidYamlValue[*SystemConfig](t, "{ \"AdditionalFiles\": { \"a.txt\": [] } }")
}

func TestSystemConfigIsValidDuplicatePartitionID(t *testing.T) {
	value := SystemConfig{
		PartitionSettings: []PartitionSetting{
			{
				ID: "a",
			},
			{
				ID: "a",
			},
		},
	}

	err := value.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "duplicate PartitionSettings ID")
}

func TestSystemConfigIsValidKernelCommandLineInvalidChars(t *testing.T) {
	value := SystemConfig{
		KernelCommandLine: KernelCommandLine{
			ExtraCommandLine: "example=\"example\"",
		},
	}

	err := value.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "ExtraCommandLine")
}

func TestSystemConfigIsValidVerityInValidPartUuid(t *testing.T) {
	invalidVerity := SystemConfig{
		Verity: &Verity{
			DataPartition: VerityPartition{
				IdType: "PartUuid",
				Id:     "incorrect-uuid-format",
			},
			HashPartition: VerityPartition{
				IdType: "PartLabel",
				Id:     "hash_partition",
			},
		},
	}

	err := invalidVerity.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid Id format")
}

func TestSystemConfigEnableGrubMkconfigConflictsVerity(t *testing.T) {
	value := SystemConfig{
		EnableGrubMkconfig: ptrutils.PtrTo(true),
		Verity: &Verity{
			DataPartition: VerityPartition{
				IdType: IdTypePartLabel,
				Id:     "a",
			},
			HashPartition: VerityPartition{
				IdType: IdTypePartLabel,
				Id:     "b",
			},
		},
	}

	err := value.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "verity")
	assert.ErrorContains(t, err, "grub2-mkconfig")
}

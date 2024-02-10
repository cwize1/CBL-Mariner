// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemConfigValidEmpty(t *testing.T) {
	testValidYamlValue[*SystemConfig](t, "{ }", &SystemConfig{})
}

func TestSystemConfigValidHostname(t *testing.T) {
	testValidYamlValue[*SystemConfig](t, "{ \"hostname\": \"validhostname\" }", &SystemConfig{Hostname: "validhostname"})
}

func TestSystemConfigInvalidHostname(t *testing.T) {
	testInvalidYamlValue[*SystemConfig](t, "{ \"hostname\": \"invalid_hostname\" }")
}

func TestSystemConfigInvalidAdditionalFiles(t *testing.T) {
	testInvalidYamlValue[*SystemConfig](t, "{ \"additionalFiles\": { \"a.txt\": [] } }")
}

func TestSystemConfigIsValidKernelCommandLineInvalidChars(t *testing.T) {
	value := SystemConfig{
		KernelCommandLine: KernelCommandLine{
			ExtraCommandLine: "example=\"example\"",
		},
	}

	err := value.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "extraCommandLine")
}

func TestSystemConfigIsValidVerityInValidPartUuid(t *testing.T) {
	invalidVerity := SystemConfig{
		Verity: &Verity{
			DataPartition: VerityPartition{
				IdType: "partuuid",
				Id:     "incorrect-uuid-format",
			},
			HashPartition: VerityPartition{
				IdType: "partlabel",
				Id:     "hash_partition",
			},
		},
	}

	err := invalidVerity.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid id format")
}

// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemConfigValidEmpty(t *testing.T) {
	testValidYamlValue[*OSConfig](t, "{ }", &OSConfig{})
}

func TestSystemConfigValidHostname(t *testing.T) {
	testValidYamlValue[*OSConfig](t, "{ \"hostname\": \"validhostname\" }", &OSConfig{Hostname: "validhostname"})
}

func TestSystemConfigInvalidHostname(t *testing.T) {
	testInvalidYamlValue[*OSConfig](t, "{ \"hostname\": \"invalid_hostname\" }")
}

func TestSystemConfigInvalidAdditionalFiles(t *testing.T) {
	testInvalidYamlValue[*OSConfig](t, "{ \"additionalFiles\": { \"a.txt\": [] } }")
}

func TestSystemConfigIsValidKernelCommandLineInvalidChars(t *testing.T) {
	value := OSConfig{
		KernelCommandLine: KernelCommandLine{
			ExtraCommandLine: "example=\"example\"",
		},
	}

	err := value.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "extraCommandLine")
}

func TestSystemConfigIsValidVerityInValidPartUuid(t *testing.T) {
	invalidVerity := OSConfig{
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

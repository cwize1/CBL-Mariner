// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystemConfigValidEmpty(t *testing.T) {
	testValidYamlValue[*OS](t, "{ }", &OS{})
}

func TestSystemConfigValidHostname(t *testing.T) {
	testValidYamlValue[*OS](t, "{ \"hostname\": \"validhostname\" }", &OS{Hostname: "validhostname"})
}

func TestSystemConfigInvalidHostname(t *testing.T) {
	testInvalidYamlValue[*OS](t, "{ \"hostname\": \"invalid_hostname\" }")
}

func TestSystemConfigInvalidAdditionalFiles(t *testing.T) {
	testInvalidYamlValue[*OS](t, "{ \"additionalFiles\": { \"a.txt\": [] } }")
}

func TestSystemConfigIsValidKernelCommandLineInvalidChars(t *testing.T) {
	value := OS{
		KernelCommandLine: KernelCommandLine{
			ExtraCommandLine: "example=\"example\"",
		},
	}

	err := value.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "extraCommandLine")
}

func TestSystemConfigIsValidVerityInValidPartUuid(t *testing.T) {
	invalidVerity := OS{
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

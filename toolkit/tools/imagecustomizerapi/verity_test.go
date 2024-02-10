// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerityIsValidInvalidDataPartition(t *testing.T) {
	invalidVerity := Verity{
		DataPartition: VerityPartition{
			IdType: "partuuid",
			Id:     "incorrect-uuid-format",
		},
		HashPartition: VerityPartition{
			IdType: "partlabel",
			Id:     "hash_partition",
		},
	}

	err := invalidVerity.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid dataPartition")
}

func TestVerityIsValidInvalidHashPartition(t *testing.T) {
	invalidVerity := Verity{
		DataPartition: VerityPartition{
			IdType: "partuuid",
			Id:     "123e4567-e89b-4d3a-a456-426614174000",
		},
		HashPartition: VerityPartition{
			IdType: "partlabel",
			Id:     "",
		},
	}

	err := invalidVerity.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid hashPartition")
}

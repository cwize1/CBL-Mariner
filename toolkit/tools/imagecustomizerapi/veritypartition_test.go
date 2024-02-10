// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerityPartitionIsValidValidPartUuidFormat(t *testing.T) {
	correctUuidPartition := VerityPartition{
		IdType: "partuuid",
		Id:     "123e4567-e89b-4d3a-a456-426614174000",
	}

	err := correctUuidPartition.IsValid()
	assert.NoError(t, err)
}

func TestVerityPartitionIsValidValidPartLabel(t *testing.T) {
	validPartition := VerityPartition{
		IdType: "partlabel",
		Id:     "ValidLabelName",
	}

	err := validPartition.IsValid()
	assert.NoError(t, err)
}

func TestVerityPartitionIsValidInvalidPartLabel(t *testing.T) {
	invalidPartition := VerityPartition{
		IdType: "partlabel",
		Id:     "",
	}

	err := invalidPartition.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid id: empty string")
}

func TestVerityPartitionIsValidInvalidEmptyPartUuid(t *testing.T) {
	emptyIdPartition := VerityPartition{
		IdType: "partuuid",
		Id:     "",
	}

	err := emptyIdPartition.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid id: empty string")
}

func TestVerityPartitionIsValidInvalidPartUuidFormat(t *testing.T) {
	incorrectUuidPartition := VerityPartition{
		IdType: "partuuid",
		Id:     "incorrect-uuid-format",
	}

	err := incorrectUuidPartition.IsValid()
	assert.Error(t, err)
	assert.ErrorContains(t, err, "invalid id format")
}

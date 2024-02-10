// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerapi

import (
	"fmt"
)

type IdType string

const (
	IdTypePartLabel IdType = "partlabel"
	IdTypeUuid      IdType = "uuid"
	IdTypePartUuid  IdType = "partuuid"
)

func (i IdType) IsValid() error {
	switch i {
	case IdTypePartLabel, IdTypeUuid, IdTypePartUuid:
		// All good.
		return nil

	default:
		return fmt.Errorf("invalid idType value (%v)", i)
	}
}

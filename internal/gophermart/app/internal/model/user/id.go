package user

import (
	"fmt"

	"github.com/google/uuid"
)

type ID uuid.UUID

func NewID() ID  { return ID(uuid.New()) }
func NillID() ID { return ID(uuid.Nil) }

func ParseID(val string) (ID, error) {
	uid, err := uuid.Parse(val)
	if err != nil {
		return ID(uuid.Nil), fmt.Errorf("ParseID [%s]: %w", val, err)
	}

	return ID(uid), nil
}

func (id ID) String() string { return uuid.UUID(id).String() }

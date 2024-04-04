package order

import "github.com/google/uuid"

type ID uuid.UUID

func ParseID(val string) (ID, error) {
	uid, err := uuid.Parse(val)
	if err != nil {
		return ID(uuid.Nil), err
	}
	return ID(uid), nil
}

func NewID() ID              { return ID(uuid.New()) }
func (id ID) String() string { return uuid.UUID(id).String() }

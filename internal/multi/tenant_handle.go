package multi

import (
	"errors"

	"github.com/google/uuid"
)

type TenantHandle struct {
	ID   uuid.UUID
	Name string
}

func ParseTenantHandle(s string) (TenantHandle, error) {
	if s == "" {
		return TenantHandle{}, errors.New("tenant handle must be set")
	}
	u, err := uuid.Parse(s)
	if err != nil {
		if err := ValidateTenantName(s); err != nil {
			return TenantHandle{}, err
		}
		return TenantHandle{Name: s}, nil
	}
	return TenantHandle{
		ID: u,
	}, nil
}

func (h TenantHandle) IsSet() bool {
	return h.ID.String() != "" || h.Name != ""
}

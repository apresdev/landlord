package authx

import (
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TenantID string

func mustNewUUID() uuid.UUID {
	u, err := uuid.NewUUID()
	if err != nil {
		zap.L().Fatal("failed to generate new UUID", zap.Error(err))
	}
	return u
}

func MustNewTenantID() TenantID {
	return TenantID(mustNewUUID().String())
}

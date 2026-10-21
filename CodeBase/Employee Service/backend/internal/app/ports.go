package app

import (
	"context"
	"time"

	"idp-platform/employee-service/internal/domain"
)

// HRISClientPort - порт к внешней кадровой системе (1С/SAP)
type HRISClientPort interface {
	FetchChanges(ctx context.Context, since time.Time) ([]domain.ChangeRecord, error)
}

// SyncStatePort - порт к хранилищу контрольной точки (checkpoint)
type SyncStatePort interface {
	GetCheckpoint(ctx context.Context) (time.Time, error)
	SetCheckpoint(ctx context.Context, checkpoint time.Time) error
}

// EmployeeServicePort - порт к Employee Management Service
type EmployeeServicePort interface {
	ProcessEmployeeUpdate(ctx context.Context, entityID string, data map[string]any) error
}

// OrganizationServicePort - порт к Organization Structure Service
type OrganizationServicePort interface {
	ProcessOrganizationUpdate(ctx context.Context, entityID string, data map[string]any) error
}

// RoleServicePort - порт к Role Management Service
type RoleServicePort interface {
	ProcessRoleUpdate(ctx context.Context, entityID string, data map[string]any) error
}


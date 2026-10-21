package strategy

import (
	"context"

	"idp-platform/employee-service/internal/domain"
)

// ChangeDispatchStrategy - интерфейс стратегии маршрутизации доменных изменений
type ChangeDispatchStrategy interface {
	TargetService() domain.TargetService
	Dispatch(ctx context.Context, change domain.ChangeRecord) error
}


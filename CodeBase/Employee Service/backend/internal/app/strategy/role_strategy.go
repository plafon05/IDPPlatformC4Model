package strategy

import (
	"context"
	"fmt"

	"idp-platform/employee-service/internal/app"
	"idp-platform/employee-service/internal/domain"
)

// RoleDispatchStrategy - стратегия передачи данных системных ролей
type RoleDispatchStrategy struct {
	port app.RoleServicePort
}

func NewRoleDispatchStrategy(port app.RoleServicePort) *RoleDispatchStrategy {
	return &RoleDispatchStrategy{port: port}
}

func (s *RoleDispatchStrategy) TargetService() domain.TargetService {
	return domain.TargetRole
}

func (s *RoleDispatchStrategy) Dispatch(ctx context.Context, change domain.ChangeRecord) error {
	fmt.Printf("  [Strategy: ROLE] Передача пересмотра роли для '%s' в RoleServicePort\n", change.EntityID)
	return s.port.ProcessRoleUpdate(ctx, change.EntityID, change.Data)
}


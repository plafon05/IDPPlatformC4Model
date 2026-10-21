package strategy

import (
	"context"
	"fmt"

	"idp-platform/employee-service/internal/app"
	"idp-platform/employee-service/internal/domain"
)

// EmployeeDispatchStrategy - стратегия передачи данных сотрудника
type EmployeeDispatchStrategy struct {
	port app.EmployeeServicePort
}

func NewEmployeeDispatchStrategy(port app.EmployeeServicePort) *EmployeeDispatchStrategy {
	return &EmployeeDispatchStrategy{port: port}
}

func (s *EmployeeDispatchStrategy) TargetService() domain.TargetService {
	return domain.TargetEmployee
}

func (s *EmployeeDispatchStrategy) Dispatch(ctx context.Context, change domain.ChangeRecord) error {
	fmt.Printf("  [Strategy: EMPLOYEE] Передача изменений сотрудника '%s' в EmployeeServicePort\n", change.EntityID)
	return s.port.ProcessEmployeeUpdate(ctx, change.EntityID, change.Data)
}


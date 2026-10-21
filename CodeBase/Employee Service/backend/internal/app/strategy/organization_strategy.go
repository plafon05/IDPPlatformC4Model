package strategy

import (
	"context"
	"fmt"

	"idp-platform/employee-service/internal/app"
	"idp-platform/employee-service/internal/domain"
)

// OrganizationDispatchStrategy - стратегия передачи данных оргструктуры
type OrganizationDispatchStrategy struct {
	port app.OrganizationServicePort
}

func NewOrganizationDispatchStrategy(port app.OrganizationServicePort) *OrganizationDispatchStrategy {
	return &OrganizationDispatchStrategy{port: port}
}

func (s *OrganizationDispatchStrategy) TargetService() domain.TargetService {
	return domain.TargetOrganization
}

func (s *OrganizationDispatchStrategy) Dispatch(ctx context.Context, change domain.ChangeRecord) error {
	fmt.Printf("  [Strategy: ORGANIZATION] Передача изменений оргструктуры '%s' в OrganizationServicePort\n", change.EntityID)
	return s.port.ProcessOrganizationUpdate(ctx, change.EntityID, change.Data)
}


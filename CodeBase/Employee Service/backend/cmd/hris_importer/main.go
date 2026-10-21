package main

import (
	"context"
	"fmt"
	"time"

	"idp-platform/employee-service/internal/app"
	"idp-platform/employee-service/internal/app/strategy"
	"idp-platform/employee-service/internal/app/usecase"
	"idp-platform/employee-service/internal/domain"
)

// ============================================================================
// Простые заглушки портов исключительно для локального запуска (demo runner)
// Внутри компонента app адаптеров нет — сервис зависит только от интерфейсов ports.go
// ============================================================================

type mockHRISClient struct{}

func (m *mockHRISClient) FetchChanges(ctx context.Context, since time.Time) ([]domain.ChangeRecord, error) {
	return []domain.ChangeRecord{
		{
			RecordID:      "1C-EMP-001",
			TargetService: domain.TargetEmployee,
			EntityID:      "EMP-1001",
			Data:          map[string]any{"first_name": "София", "last_name": "Жбанова", "grade": "Senior"},
			Timestamp:     time.Now().Add(-10 * time.Minute),
		},
		{
			RecordID:      "1C-ORG-002",
			TargetService: domain.TargetOrganization,
			EntityID:      "DEP-404",
			Data:          map[string]any{"dept_name": "Платформенная разработка", "head_id": "EMP-1001"},
			Timestamp:     time.Now().Add(-5 * time.Minute),
		},
		{
			RecordID:      "1C-ROLE-003",
			TargetService: domain.TargetRole,
			EntityID:      "EMP-1001",
			Data:          map[string]any{"role": "TeamLead", "scope": "IDP_Core"},
			Timestamp:     time.Now(),
		},
	}, nil
}

type mockSyncState struct {
	checkpoint time.Time
}

func (m *mockSyncState) GetCheckpoint(ctx context.Context) (time.Time, error) {
	return m.checkpoint, nil
}

func (m *mockSyncState) SetCheckpoint(ctx context.Context, checkpoint time.Time) error {
	m.checkpoint = checkpoint
	return nil
}

type mockEmployeeServicePort struct{}

func (m *mockEmployeeServicePort) ProcessEmployeeUpdate(ctx context.Context, entityID string, data map[string]any) error {
	fmt.Printf("    -> [Employee Service Port] Обновлен сотрудник %s: %+v\n", entityID, data)
	return nil
}

type mockOrganizationServicePort struct{}

func (m *mockOrganizationServicePort) ProcessOrganizationUpdate(ctx context.Context, entityID string, data map[string]any) error {
	fmt.Printf("    -> [Organization Service Port] Обновлен отдел %s: %+v\n", entityID, data)
	return nil
}

type mockRoleServicePort struct{}

func (m *mockRoleServicePort) ProcessRoleUpdate(ctx context.Context, entityID string, data map[string]any) error {
	fmt.Printf("    -> [Role Service Port] Назначена роль для %s: %+v\n", entityID, data)
	return nil
}

func main() {
	ctx := context.Background()

	// 1. Инициализация портов
	hrisClient := &mockHRISClient{}
	syncState := &mockSyncState{checkpoint: time.Now().Add(-1 * time.Hour)}
	empPort := &mockEmployeeServicePort{}
	orgPort := &mockOrganizationServicePort{}
	rolePort := &mockRoleServicePort{}

	// 2. Инициализация реестра стратегий (папка strategy)
	registry := strategy.NewStrategyRegistry()
	registry.Register(strategy.NewEmployeeDispatchStrategy(empPort))
	registry.Register(strategy.NewOrganizationDispatchStrategy(orgPort))
	registry.Register(strategy.NewRoleDispatchStrategy(rolePort))

	// 3. Инициализация Use Cases (папка usecase)
	fetchUC := usecase.NewFetchHRISDataUseCase(syncState, hrisClient)
	applyUC := usecase.NewApplyHRISDataUseCase(registry, syncState)

	// 4. Инициализация оркестратора HRISImportService
	importService := app.NewHRISImportService(fetchUC, applyUC)

	// 5. Запуск цикла импорта
	if err := importService.RunImportCycle(ctx); err != nil {
		fmt.Printf("Ошибка выполнения цикла импорта: %v\n", err)
	}
}

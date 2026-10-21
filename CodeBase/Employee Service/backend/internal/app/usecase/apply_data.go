package usecase

import (
	"context"
	"fmt"
	"time"

	"idp-platform/employee-service/internal/app"
	"idp-platform/employee-service/internal/app/strategy"
	"idp-platform/employee-service/internal/domain"
)

// ApplyHRISDataUseCase - юзкейс 2: оркестрация применения через Стратегии
type ApplyHRISDataUseCase struct {
	registry  *strategy.StrategyRegistry
	syncState app.SyncStatePort
}

func NewApplyHRISDataUseCase(registry *strategy.StrategyRegistry, syncState app.SyncStatePort) *ApplyHRISDataUseCase {
	return &ApplyHRISDataUseCase{
		registry:  registry,
		syncState: syncState,
	}
}

func (uc *ApplyHRISDataUseCase) Execute(ctx context.Context, changes []domain.ChangeRecord) error {
	fmt.Printf("\n[UseCase 2: Apply] Диспетчеризация %d записей через Стратегии:\n", len(changes))
	var latestTimestamp time.Time

	for _, change := range changes {
		strat, err := uc.registry.Get(change.TargetService)
		if err != nil {
			return err
		}

		if err := strat.Dispatch(ctx, change); err != nil {
			return fmt.Errorf("ошибка применения изменения %s: %w", change.RecordID, err)
		}

		if change.Timestamp.After(latestTimestamp) {
			latestTimestamp = change.Timestamp
		}
	}

	if !latestTimestamp.IsZero() {
		if err := uc.syncState.SetCheckpoint(ctx, latestTimestamp); err != nil {
			return fmt.Errorf("ошибка фиксации чекпоинта: %w", err)
		}
		fmt.Printf("[UseCase 2: Apply] Контрольная точка успешно обновлена: %s\n", latestTimestamp.Format("2006-01-02 15:04:05"))
	}

	return nil
}


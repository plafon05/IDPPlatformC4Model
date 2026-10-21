package usecase

import (
	"context"
	"fmt"

	"idp-platform/employee-service/internal/app"
	"idp-platform/employee-service/internal/domain"
)

// FetchHRISDataUseCase - юзкейс 1: опрос внешнего клиента и получение изменений
type FetchHRISDataUseCase struct {
	syncState  app.SyncStatePort
	hrisClient app.HRISClientPort
}

func NewFetchHRISDataUseCase(syncState app.SyncStatePort, hrisClient app.HRISClientPort) *FetchHRISDataUseCase {
	return &FetchHRISDataUseCase{
		syncState:  syncState,
		hrisClient: hrisClient,
	}
}

func (uc *FetchHRISDataUseCase) Execute(ctx context.Context) ([]domain.ChangeRecord, error) {
	checkpoint, err := uc.syncState.GetCheckpoint(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения чекпоинта: %w", err)
	}

	fmt.Printf("[UseCase 1: Fetch] Запрос изменений с контрольной точки: %s\n", checkpoint.Format("2006-01-02 15:04:05"))
	return uc.hrisClient.FetchChanges(ctx, checkpoint)
}


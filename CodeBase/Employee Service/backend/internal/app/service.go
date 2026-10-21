package app

import (
	"context"
	"fmt"

	"idp-platform/employee-service/internal/domain"
)

// FetchDataUseCase - интерфейс сценария получения данных
type FetchDataUseCase interface {
	Execute(ctx context.Context) ([]domain.ChangeRecord, error)
}

// ApplyDataUseCase - интерфейс сценария применения данных
type ApplyDataUseCase interface {
	Execute(ctx context.Context, changes []domain.ChangeRecord) error
}

// HRISImportService - прикладной сервис оркестрации полного цикла импорта
// Запускается фоновым воркером HRIS Polling Worker через прямой Go-вызов
type HRISImportService struct {
	fetchUC FetchDataUseCase
	applyUC ApplyDataUseCase
}

func NewHRISImportService(fetchUC FetchDataUseCase, applyUC ApplyDataUseCase) *HRISImportService {
	return &HRISImportService{
		fetchUC: fetchUC,
		applyUC: applyUC,
	}
}

func (s *HRISImportService) RunImportCycle(ctx context.Context) error {
	fmt.Println("======================================================================")
	fmt.Println("HRIS IMPORT SERVICE (Go): ЗАПУСК ЦИКЛА ВЫГРУЗКИ КАДРОВЫХ ДАННЫХ")
	fmt.Println("======================================================================")

	changes, err := s.fetchUC.Execute(ctx)
	if err != nil {
		return err
	}

	if len(changes) == 0 {
		fmt.Println("Нет новых кадровых изменений.")
		return nil
	}

	if err := s.applyUC.Execute(ctx, changes); err != nil {
		return err
	}

	fmt.Println("======================================================================")
	fmt.Println("ЦИКЛ СИНХРОНИЗАЦИИ УСПЕШНО ЗАВЕРШЁН")
	fmt.Println("======================================================================")
	return nil
}

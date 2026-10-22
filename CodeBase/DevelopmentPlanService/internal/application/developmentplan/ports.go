package developmentplan

import (
	"context"

	"idp-platform/development-plan-service/internal/domain/plan"
)

// PlanRepository — порт к Plan Repository (L3). Реализация на pgx в модуль не входит.
type PlanRepository interface {
	Get(ctx context.Context, id string) (plan.Plan, error)
	// Save сохраняет план, если его версия всё ещё равна expectedVersion,
	// и пишет события в outbox в той же транзакции.
	Save(ctx context.Context, p plan.Plan, expectedVersion int, events []plan.TaskStatusChanged) error
}

// ResultSource — откуда пришёл результат задания.
type ResultSource string

const (
	SourceEmployee ResultSource = "employee" // отметка сотрудника через Development API Controller
	SourceLMS      ResultSource = "lms"      // результат курса LMS (событие Talent Service из Kafka)
	SourceTest     ResultSource = "test"     // результат теста (событие Talent Service из Kafka)
)

// TaskResult — результат выполнения задания.
type TaskResult struct {
	Source          ResultSource
	AttachmentIDs   []string // подтверждающие материалы в Object Storage
	CourseCompleted bool     // для LMS: курс пройден
	Score           *float64 // для теста: итоговый балл; nil — баллы ещё не подсчитаны (ручная проверка)
}

// Actor — инициатор операции: пользователь из токена или системная интеграция.
type Actor struct {
	UserID string
	System bool
}

// SubmitResultCommand — фиксация результата задания.
type SubmitResultCommand struct {
	PlanID string
	TaskID string
	Actor  Actor
	Result TaskResult
}

// ReviewTaskCommand — решение руководителя по заданию с проверкой.
type ReviewTaskCommand struct {
	PlanID  string
	TaskID  string
	Actor   Actor
	Approve bool
	Comment string
}

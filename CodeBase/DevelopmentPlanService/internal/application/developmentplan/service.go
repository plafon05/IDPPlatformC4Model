package developmentplan

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"idp-platform/development-plan-service/internal/domain/plan"
)

var (
	ErrForbidden       = errors.New("нет прав на операцию с заданием ИПР")
	ErrInvalidInput    = errors.New("некорректная команда")
	ErrCommentRequired = errors.New("при возврате на доработку нужен комментарий")
)

// DevelopmentPlanService — контекст паттерна «Стратегия».
// Общий сценарий (права, состояние ИПР, переход задания, прогресс, outbox) одинаков,
// а проверка критерия выполнения подставляется стратегией по task.Criterion.
type DevelopmentPlanService struct {
	plans    PlanRepository
	registry *StrategyRegistry
	now      func() time.Time
}

func NewDevelopmentPlanService(plans PlanRepository, registry *StrategyRegistry) (*DevelopmentPlanService, error) {
	if plans == nil || registry == nil {
		return nil, errors.New("не все зависимости DevelopmentPlanService заданы")
	}
	return &DevelopmentPlanService{plans: plans, registry: registry, now: time.Now}, nil
}

// SubmitResult фиксирует результат задания и пересчитывает взвешенный прогресс ИПР.
func (s *DevelopmentPlanService) SubmitResult(ctx context.Context, cmd SubmitResultCommand) (plan.Plan, error) {
	p, task, err := s.load(ctx, cmd.PlanID, cmd.TaskID)
	if err != nil {
		return plan.Plan{}, err
	}
	if !canSubmit(cmd.Actor, p, cmd.Result.Source) {
		return plan.Plan{}, ErrForbidden
	}

	strategy, err := s.registry.Resolve(task.Criterion)
	if err != nil {
		return plan.Plan{}, err
	}
	met, err := strategy.IsMet(*task, cmd.Result)
	if err != nil {
		return plan.Plan{}, err
	}

	from := task.Status
	if met {
		err = task.Accept()
	} else {
		err = task.Start()
	}
	if err != nil {
		return plan.Plan{}, err
	}
	return s.save(ctx, p, task, from)
}

// ReviewTask — руководитель подтверждает результат или возвращает задание на доработку.
func (s *DevelopmentPlanService) ReviewTask(ctx context.Context, cmd ReviewTaskCommand) (plan.Plan, error) {
	p, task, err := s.load(ctx, cmd.PlanID, cmd.TaskID)
	if err != nil {
		return plan.Plan{}, err
	}
	if cmd.Actor.System || cmd.Actor.UserID != p.ManagerID {
		return plan.Plan{}, ErrForbidden
	}

	from := task.Status
	if cmd.Approve {
		err = task.Confirm()
	} else if strings.TrimSpace(cmd.Comment) == "" {
		err = ErrCommentRequired
	} else {
		err = task.Return()
	}
	if err != nil {
		return plan.Plan{}, err
	}
	return s.save(ctx, p, task, from)
}

func (s *DevelopmentPlanService) load(ctx context.Context, planID, taskID string) (*plan.Plan, *plan.Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if planID == "" || taskID == "" {
		return nil, nil, ErrInvalidInput
	}
	p, err := s.plans.Get(ctx, planID)
	if err != nil {
		return nil, nil, err
	}
	if p.Status != plan.StatusActive {
		return nil, nil, plan.ErrPlanNotActive
	}
	task, err := p.Task(taskID)
	if err != nil {
		return nil, nil, err
	}
	return &p, task, nil
}

func (s *DevelopmentPlanService) save(ctx context.Context, p *plan.Plan, task *plan.Task, from plan.TaskStatus) (plan.Plan, error) {
	if task.Status == from {
		return *p, nil // состояние не изменилось — событие не публикуется
	}
	event := plan.TaskStatusChanged{
		PlanID:     p.ID,
		TaskID:     task.ID,
		EmployeeID: p.EmployeeID,
		ManagerID:  p.ManagerID,
		From:       from,
		To:         task.Status,
		Progress:   p.Progress(),
		OccurredAt: s.now().UTC(),
	}
	expected := p.Version
	p.Version++
	if err := s.plans.Save(ctx, *p, expected, []plan.TaskStatusChanged{event}); err != nil {
		return plan.Plan{}, fmt.Errorf("сохранение ИПР %s: %w", p.ID, err)
	}
	return *p, nil
}

// canSubmit: отметку ставит только сотрудник — владелец ИПР;
// результаты LMS и тестов принимаются только от системной интеграции.
func canSubmit(actor Actor, p *plan.Plan, source ResultSource) bool {
	if source == SourceEmployee {
		return !actor.System && actor.UserID == p.EmployeeID
	}
	return actor.System
}

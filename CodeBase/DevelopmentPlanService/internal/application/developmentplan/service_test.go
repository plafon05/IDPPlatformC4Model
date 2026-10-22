package developmentplan

import (
	"context"
	"errors"
	"testing"
	"time"

	"idp-platform/development-plan-service/internal/domain/plan"
)

// Тестовые заглушки портов. Это не инфраструктурные адаптеры.
type plansStub struct {
	item   plan.Plan
	events []plan.TaskStatusChanged
}

func (s *plansStub) Get(_ context.Context, id string) (plan.Plan, error) {
	if id != s.item.ID {
		return plan.Plan{}, errors.New("not found")
	}
	p := s.item
	p.Tasks = append([]plan.Task(nil), s.item.Tasks...)
	return p, nil
}

func (s *plansStub) Save(_ context.Context, p plan.Plan, expected int, events []plan.TaskStatusChanged) error {
	if s.item.Version != expected {
		return errors.New("conflict")
	}
	s.item = p
	s.events = append(s.events, events...)
	return nil
}

type countingStrategy struct{ calls *int }

func (s countingStrategy) Criterion() plan.Criterion { return plan.CriterionEmployeeMark }
func (s countingStrategy) IsMet(plan.Task, TaskResult) (bool, error) {
	*s.calls++
	return false, nil
}

var (
	employee = Actor{UserID: "e1"}
	manager  = Actor{UserID: "m1"}
	system   = Actor{System: true}
)

func score(v float64) *float64 { return &v }

func newPlans() *plansStub {
	return &plansStub{item: plan.Plan{
		ID: "p1", EmployeeID: "e1", ManagerID: "m1", Status: plan.StatusActive,
		Tasks: []plan.Task{
			{ID: "read", Weight: 1, Criterion: plan.CriterionEmployeeMark, Status: plan.TaskNotStarted},
			{ID: "practice", Weight: 2, Criterion: plan.CriterionEmployeeMark, RequiresReview: true, Status: plan.TaskInProgress},
			{ID: "course", Weight: 3, Criterion: plan.CriterionLMSCourse, Status: plan.TaskNotStarted},
			{ID: "test", Weight: 4, Criterion: plan.CriterionTestResult, PassingScore: score(70), Status: plan.TaskNotStarted},
		},
	}}
}

func newService(t *testing.T, plans *plansStub) *DevelopmentPlanService {
	t.Helper()
	registry := NewStrategyRegistry()
	for _, s := range []CompletionStrategy{NewEmployeeMarkStrategy(), NewLMSCourseStrategy(), NewTestResultStrategy()} {
		if err := registry.Register(s); err != nil {
			t.Fatal(err)
		}
	}
	svc, err := NewDevelopmentPlanService(plans, registry)
	if err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC) }
	return svc
}

func status(p plan.Plan, id string) plan.TaskStatus {
	task, _ := p.Task(id)
	return task.Status
}

func TestEmployeeMarkCompletesTaskAndRecalculatesProgress(t *testing.T) {
	plans := newPlans()
	got, err := newService(t, plans).SubmitResult(context.Background(), SubmitResultCommand{
		PlanID: "p1", TaskID: "read", Actor: employee, Result: TaskResult{Source: SourceEmployee},
	})
	if err != nil || status(got, "read") != plan.TaskCompleted {
		t.Fatalf("plan=%+v err=%v", got, err)
	}
	if len(plans.events) != 1 || *plans.events[0].Progress != 10 { // 1 / (1+2+3+4)
		t.Fatalf("неверное событие: %+v", plans.events)
	}
}

func TestTaskWithReviewGoesToManager(t *testing.T) {
	plans := newPlans()
	svc := newService(t, plans)
	got, err := svc.SubmitResult(context.Background(), SubmitResultCommand{
		PlanID: "p1", TaskID: "practice", Actor: employee, Result: TaskResult{Source: SourceEmployee, AttachmentIDs: []string{"f1"}},
	})
	if err != nil || status(got, "practice") != plan.TaskOnReview || *got.Progress() != 0 {
		t.Fatalf("plan=%+v err=%v", got, err)
	}
	got, err = svc.ReviewTask(context.Background(), ReviewTaskCommand{PlanID: "p1", TaskID: "practice", Actor: manager, Approve: true})
	if err != nil || status(got, "practice") != plan.TaskCompleted || *got.Progress() != 20 {
		t.Fatalf("plan=%+v err=%v", got, err)
	}
}

func TestManagerReturnRequiresComment(t *testing.T) {
	plans := newPlans()
	plans.item.Tasks[1].Status = plan.TaskOnReview
	svc := newService(t, plans)
	_, err := svc.ReviewTask(context.Background(), ReviewTaskCommand{PlanID: "p1", TaskID: "practice", Actor: manager})
	if !errors.Is(err, ErrCommentRequired) {
		t.Fatalf("ожидался ErrCommentRequired, получено %v", err)
	}
	got, err := svc.ReviewTask(context.Background(), ReviewTaskCommand{PlanID: "p1", TaskID: "practice", Actor: manager, Comment: "Добавь ссылку на PR"})
	if err != nil || status(got, "practice") != plan.TaskInProgress {
		t.Fatalf("plan=%+v err=%v", got, err)
	}
}

func TestLMSCourseAcceptsOnlyLMSResult(t *testing.T) {
	svc := newService(t, newPlans())
	_, err := svc.SubmitResult(context.Background(), SubmitResultCommand{
		PlanID: "p1", TaskID: "course", Actor: employee, Result: TaskResult{Source: SourceEmployee},
	})
	if !errors.Is(err, ErrResultNotAccepted) {
		t.Fatalf("ожидался ErrResultNotAccepted, получено %v", err)
	}
	got, err := svc.SubmitResult(context.Background(), SubmitResultCommand{
		PlanID: "p1", TaskID: "course", Actor: system, Result: TaskResult{Source: SourceLMS, CourseCompleted: true},
	})
	if err != nil || status(got, "course") != plan.TaskCompleted {
		t.Fatalf("plan=%+v err=%v", got, err)
	}
}

func TestTestResultBelowPassingScoreKeepsTaskInProgress(t *testing.T) {
	plans := newPlans()
	got, err := newService(t, plans).SubmitResult(context.Background(), SubmitResultCommand{
		PlanID: "p1", TaskID: "test", Actor: system, Result: TaskResult{Source: SourceTest, Score: score(55)},
	})
	if err != nil || status(got, "test") != plan.TaskInProgress {
		t.Fatalf("plan=%+v err=%v", got, err)
	}
}

func TestTestResultWaitsForManualCheck(t *testing.T) {
	plans := newPlans()
	plans.item.Tasks[3].Status = plan.TaskInProgress
	got, err := newService(t, plans).SubmitResult(context.Background(), SubmitResultCommand{
		PlanID: "p1", TaskID: "test", Actor: system, Result: TaskResult{Source: SourceTest},
	})
	if err != nil || status(got, "test") != plan.TaskInProgress || len(plans.events) != 0 {
		t.Fatalf("без подсчёта баллов задание не завершается: plan=%+v err=%v", got, err)
	}
}

func TestTestResultReachingScoreCompletesTask(t *testing.T) {
	got, err := newService(t, newPlans()).SubmitResult(context.Background(), SubmitResultCommand{
		PlanID: "p1", TaskID: "test", Actor: system, Result: TaskResult{Source: SourceTest, Score: score(70)},
	})
	if err != nil || status(got, "test") != plan.TaskCompleted || *got.Progress() != 40 {
		t.Fatalf("plan=%+v err=%v", got, err)
	}
}

func TestFullProgressDoesNotCompletePlan(t *testing.T) {
	plans := newPlans()
	plans.item.Tasks = plans.item.Tasks[:1]
	got, err := newService(t, plans).SubmitResult(context.Background(), SubmitResultCommand{
		PlanID: "p1", TaskID: "read", Actor: employee, Result: TaskResult{Source: SourceEmployee},
	})
	if err != nil || *got.Progress() != 100 || got.Status != plan.StatusActive {
		t.Fatalf("100%% не должно завершать ИПР: plan=%+v err=%v", got, err)
	}
}

func TestOnlyPlanOwnerCanMarkTask(t *testing.T) {
	_, err := newService(t, newPlans()).SubmitResult(context.Background(), SubmitResultCommand{
		PlanID: "p1", TaskID: "read", Actor: Actor{UserID: "e2"}, Result: TaskResult{Source: SourceEmployee},
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("ожидался ErrForbidden, получено %v", err)
	}
}

func TestDraftPlanRejectsResults(t *testing.T) {
	plans := newPlans()
	plans.item.Status = plan.StatusDraft
	_, err := newService(t, plans).SubmitResult(context.Background(), SubmitResultCommand{
		PlanID: "p1", TaskID: "read", Actor: employee, Result: TaskResult{Source: SourceEmployee},
	})
	if !errors.Is(err, plan.ErrPlanNotActive) {
		t.Fatalf("ожидался ErrPlanNotActive, получено %v", err)
	}
}

// Подмена стратегии в реестре меняет поведение без изменения DevelopmentPlanService.
func TestReplaceStrategy(t *testing.T) {
	plans := newPlans()
	svc := newService(t, plans)
	calls := 0
	_ = svc.registry.Register(countingStrategy{calls: &calls})
	got, err := svc.SubmitResult(context.Background(), SubmitResultCommand{
		PlanID: "p1", TaskID: "read", Actor: employee, Result: TaskResult{Source: SourceEmployee},
	})
	if err != nil || calls != 1 || status(got, "read") != plan.TaskInProgress {
		t.Fatalf("calls=%d plan=%+v err=%v", calls, got, err)
	}
}

package main

import (
	"context"
	"errors"
	"fmt"

	"idp-platform/development-plan-service/internal/application/developmentplan"
	"idp-platform/development-plan-service/internal/domain/plan"
)

// ============================================================================
// Заглушка порта только для локального запуска (demo runner).
// Сам компонент зависит лишь от интерфейса PlanRepository из ports.go.
// ============================================================================

type memoryPlans struct{ item plan.Plan }

func (m *memoryPlans) Get(_ context.Context, id string) (plan.Plan, error) {
	if id != m.item.ID {
		return plan.Plan{}, errors.New("ИПР не найден")
	}
	p := m.item
	p.Tasks = append([]plan.Task(nil), m.item.Tasks...)
	return p, nil
}

func (m *memoryPlans) Save(_ context.Context, p plan.Plan, expected int, events []plan.TaskStatusChanged) error {
	if m.item.Version != expected {
		return errors.New("конфликт версий")
	}
	m.item = p
	for _, e := range events {
		fmt.Printf("    -> [Plan Repository] задание %s: %s -> %s, прогресс %s; событие записано в outbox\n",
			e.TaskID, e.From, e.To, percent(e.Progress))
	}
	return nil
}

func percent(v *int) string {
	if v == nil {
		return "не рассчитан"
	}
	return fmt.Sprintf("%d%%", *v)
}

func main() {
	ctx := context.Background()
	passing := 70.0

	// 1. Порт
	plans := &memoryPlans{item: plan.Plan{
		ID: "IDP-1", EmployeeID: "EMP-1", ManagerID: "MGR-1", Status: plan.StatusActive,
		Tasks: []plan.Task{
			{ID: "materials", Title: "Изучить гайд по Go", Weight: 1, Criterion: plan.CriterionEmployeeMark, Status: plan.TaskNotStarted},
			{ID: "practice", Title: "Сервис на Go с ревью", Weight: 3, Criterion: plan.CriterionEmployeeMark, RequiresReview: true, Status: plan.TaskNotStarted},
			{ID: "course", Title: "Курс LMS «Конкурентность в Go»", Weight: 2, Criterion: plan.CriterionLMSCourse, Status: plan.TaskNotStarted},
			{ID: "test", Title: "Тест по этапу 1", Weight: 2, Criterion: plan.CriterionTestResult, PassingScore: &passing, Status: plan.TaskNotStarted},
		},
	}}

	// 2. Реестр стратегий
	registry := developmentplan.NewStrategyRegistry()
	_ = registry.Register(developmentplan.NewEmployeeMarkStrategy())
	_ = registry.Register(developmentplan.NewLMSCourseStrategy())
	_ = registry.Register(developmentplan.NewTestResultStrategy())

	// 3. Контекст стратегии
	service, _ := developmentplan.NewDevelopmentPlanService(plans, registry)
	employee := developmentplan.Actor{UserID: "EMP-1"}
	manager := developmentplan.Actor{UserID: "MGR-1"}
	system := developmentplan.Actor{System: true}
	score := func(v float64) *float64 { return &v }

	submit := func(title string, cmd developmentplan.SubmitResultCommand) {
		fmt.Println("[DevelopmentPlanService.SubmitResult]", title)
		report(service.SubmitResult(ctx, cmd))
	}

	submit("сотрудник отмечает изучение материалов", developmentplan.SubmitResultCommand{
		PlanID: "IDP-1", TaskID: "materials", Actor: employee,
		Result: developmentplan.TaskResult{Source: developmentplan.SourceEmployee}})
	submit("сотрудник прикладывает результат практики", developmentplan.SubmitResultCommand{
		PlanID: "IDP-1", TaskID: "practice", Actor: employee,
		Result: developmentplan.TaskResult{Source: developmentplan.SourceEmployee, AttachmentIDs: []string{"repo-link.txt"}}})

	fmt.Println("[DevelopmentPlanService.ReviewTask] руководитель подтверждает практику")
	report(service.ReviewTask(ctx, developmentplan.ReviewTaskCommand{PlanID: "IDP-1", TaskID: "practice", Actor: manager, Approve: true}))

	submit("сотрудник пытается сам отметить курс LMS", developmentplan.SubmitResultCommand{
		PlanID: "IDP-1", TaskID: "course", Actor: employee,
		Result: developmentplan.TaskResult{Source: developmentplan.SourceEmployee}})
	submit("из LMS пришло завершение курса", developmentplan.SubmitResultCommand{
		PlanID: "IDP-1", TaskID: "course", Actor: system,
		Result: developmentplan.TaskResult{Source: developmentplan.SourceLMS, CourseCompleted: true}})
	submit("результат теста 55 из 100 (нужно 70)", developmentplan.SubmitResultCommand{
		PlanID: "IDP-1", TaskID: "test", Actor: system,
		Result: developmentplan.TaskResult{Source: developmentplan.SourceTest, Score: score(55)}})
	submit("повторная попытка теста: 82 из 100", developmentplan.SubmitResultCommand{
		PlanID: "IDP-1", TaskID: "test", Actor: system,
		Result: developmentplan.TaskResult{Source: developmentplan.SourceTest, Score: score(82)}})

	fmt.Printf("Итог: прогресс %s, статус ИПР %q — завершает план руководитель отдельным решением\n",
		percent(plans.item.Progress()), plans.item.Status)
}

func report(p plan.Plan, err error) {
	if err != nil {
		fmt.Printf("    ! отклонено: %v\n", err)
		return
	}
	fmt.Printf("    = прогресс ИПР: %s\n", percent(p.Progress()))
}

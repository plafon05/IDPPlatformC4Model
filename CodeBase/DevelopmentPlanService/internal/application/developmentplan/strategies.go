package developmentplan

import (
	"errors"
	"fmt"

	"idp-platform/development-plan-service/internal/domain/plan"
)

var ErrResultNotAccepted = errors.New("результат не подходит для критерия задания")

// EmployeeMarkStrategy — изучение материалов, практика, самостоятельная работа:
// критерий выполнен отметкой сотрудника (материалы прикладываются по желанию).
type EmployeeMarkStrategy struct{}

func NewEmployeeMarkStrategy() *EmployeeMarkStrategy { return &EmployeeMarkStrategy{} }

func (s *EmployeeMarkStrategy) Criterion() plan.Criterion { return plan.CriterionEmployeeMark }

func (s *EmployeeMarkStrategy) IsMet(_ plan.Task, result TaskResult) (bool, error) {
	if result.Source != SourceEmployee {
		return false, rejected(s.Criterion(), result.Source)
	}
	return true, nil
}

// LMSCourseStrategy — курс внешней LMS: учитывается только достоверный результат из LMS,
// отметка сотрудника не принимается. Уровень компетенции не меняется.
type LMSCourseStrategy struct{}

func NewLMSCourseStrategy() *LMSCourseStrategy { return &LMSCourseStrategy{} }

func (s *LMSCourseStrategy) Criterion() plan.Criterion { return plan.CriterionLMSCourse }

func (s *LMSCourseStrategy) IsMet(_ plan.Task, result TaskResult) (bool, error) {
	if result.Source != SourceLMS {
		return false, rejected(s.Criterion(), result.Source)
	}
	return result.CourseCompleted, nil
}

// TestResultStrategy — тестовое задание: прохождение теста или достижение PassingScore.
// Пока баллы не подсчитаны (идёт ручная проверка открытых ответов), критерий не выполнен.
type TestResultStrategy struct{}

func NewTestResultStrategy() *TestResultStrategy { return &TestResultStrategy{} }

func (s *TestResultStrategy) Criterion() plan.Criterion { return plan.CriterionTestResult }

func (s *TestResultStrategy) IsMet(task plan.Task, result TaskResult) (bool, error) {
	if result.Source != SourceTest {
		return false, rejected(s.Criterion(), result.Source)
	}
	if result.Score == nil {
		return false, nil
	}
	if task.PassingScore == nil {
		return true, nil
	}
	return *result.Score >= *task.PassingScore, nil
}

func rejected(criterion plan.Criterion, source ResultSource) error {
	return fmt.Errorf("%w: критерий %q, источник %q", ErrResultNotAccepted, criterion, source)
}

var (
	_ CompletionStrategy = (*EmployeeMarkStrategy)(nil)
	_ CompletionStrategy = (*LMSCourseStrategy)(nil)
	_ CompletionStrategy = (*TestResultStrategy)(nil)
)

package plan

import (
	"errors"
	"fmt"
)

// TaskStatus — статус задания ИПР.
type TaskStatus string

const (
	TaskNotStarted TaskStatus = "not_started"
	TaskInProgress TaskStatus = "in_progress"
	TaskOnReview   TaskStatus = "on_review" // результат отправлен руководителю на проверку
	TaskCompleted  TaskStatus = "completed"
)

// Criterion — критерий выполнения задания, заданный руководителем в ПР/ИПР.
// По нему DevelopmentPlanService выбирает стратегию проверки результата.
type Criterion string

const (
	CriterionEmployeeMark Criterion = "employee_mark" // отметка сотрудника (изучение материалов, практика)
	CriterionLMSCourse    Criterion = "lms_course"    // прохождение курса во внешней LMS
	CriterionTestResult   Criterion = "test_result"   // прохождение теста или достижение результата
)

var (
	ErrInvalidTaskTransition = errors.New("недопустимое изменение статуса задания")
	ErrInvalidWeight         = errors.New("вес задания должен быть положительным")
)

// Task — задание ИПР.
type Task struct {
	ID             string
	Title          string
	Weight         int
	Criterion      Criterion
	RequiresReview bool     // нужно подтверждение руководителя
	PassingScore   *float64 // для теста: требуемый результат; nil — достаточно прохождения
	Status         TaskStatus
}

// Start фиксирует, что работа над заданием начата, но критерий ещё не выполнен.
func (t *Task) Start() error {
	switch t.Status {
	case TaskNotStarted:
		t.Status = TaskInProgress
		return nil
	case TaskInProgress:
		return nil
	default:
		return t.transitionError(TaskInProgress)
	}
}

// Accept применяется, когда критерий выполнения достигнут:
// задание без проверки завершается, задание с проверкой уходит руководителю.
func (t *Task) Accept() error {
	if t.Status != TaskNotStarted && t.Status != TaskInProgress {
		return t.transitionError(TaskCompleted)
	}
	if t.RequiresReview {
		t.Status = TaskOnReview
	} else {
		t.Status = TaskCompleted
	}
	return nil
}

// Confirm — руководитель подтверждает результат задания с проверкой.
func (t *Task) Confirm() error {
	if t.Status != TaskOnReview {
		return t.transitionError(TaskCompleted)
	}
	t.Status = TaskCompleted
	return nil
}

// Return — руководитель возвращает задание на доработку.
func (t *Task) Return() error {
	if t.Status != TaskOnReview {
		return t.transitionError(TaskInProgress)
	}
	t.Status = TaskInProgress
	return nil
}

func (t *Task) transitionError(to TaskStatus) error {
	return fmt.Errorf("%w: задание %s, %s -> %s", ErrInvalidTaskTransition, t.ID, t.Status, to)
}

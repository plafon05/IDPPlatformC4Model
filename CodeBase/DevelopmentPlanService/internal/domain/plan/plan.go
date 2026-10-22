package plan

import (
	"errors"
	"math"
	"time"
)

// Status — базовые состояния ИПР.
type Status string

const (
	StatusDraft     Status = "draft"     // черновик
	StatusActive    Status = "active"    // выполняется
	StatusCompleted Status = "completed" // завершён (только решением руководителя)
	StatusCancelled Status = "cancelled" // отменён
)

var (
	ErrPlanNotActive = errors.New("ИПР не находится в состоянии «выполняется»")
	ErrTaskNotFound  = errors.New("задание не найдено в ИПР")
)

// Plan — агрегат ИПР с актуальным составом заданий.
type Plan struct {
	ID         string
	EmployeeID string
	ManagerID  string
	Status     Status
	Version    int // для оптимистичной блокировки при сохранении
	Tasks      []Task
}

// Progress — взвешенный процент выполнения:
// Σ весов выполненных заданий / Σ весов всех актуальных заданий × 100%.
// Если заданий нет, процент не рассчитывается (nil).
// 100% не переводит ИПР в «Завершён» и не подтверждает освоение компетенций.
func (p Plan) Progress() *int {
	total, done := 0, 0
	for _, t := range p.Tasks {
		total += t.Weight
		if t.Status == TaskCompleted {
			done += t.Weight
		}
	}
	if total == 0 {
		return nil
	}
	value := int(math.Round(float64(done) * 100 / float64(total)))
	return &value
}

// Task возвращает указатель на задание для изменения внутри агрегата.
func (p *Plan) Task(id string) (*Task, error) {
	for i := range p.Tasks {
		if p.Tasks[i].ID == id {
			return &p.Tasks[i], nil
		}
	}
	return nil, ErrTaskNotFound
}

// AddTask и RemoveTask меняют состав заданий; Progress() сразу учитывает новый состав,
// а выполненные ранее задания сохраняют свой статус.
func (p *Plan) AddTask(t Task) error {
	if t.Weight <= 0 {
		return ErrInvalidWeight
	}
	if t.Status == "" {
		t.Status = TaskNotStarted
	}
	p.Tasks = append(p.Tasks, t)
	return nil
}

func (p *Plan) RemoveTask(id string) error {
	for i := range p.Tasks {
		if p.Tasks[i].ID == id {
			p.Tasks = append(p.Tasks[:i], p.Tasks[i+1:]...)
			return nil
		}
	}
	return ErrTaskNotFound
}

// TaskStatusChanged — доменное событие для outbox. По нему Notification Service
// уведомляет об отправке результата на проверку, Analytics обновляет прогресс.
type TaskStatusChanged struct {
	PlanID     string
	TaskID     string
	EmployeeID string
	ManagerID  string
	From       TaskStatus
	To         TaskStatus
	Progress   *int
	OccurredAt time.Time
}

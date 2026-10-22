package plan

import (
	"errors"
	"testing"
)

func progress(t *testing.T, p Plan) int {
	t.Helper()
	v := p.Progress()
	if v == nil {
		t.Fatal("прогресс не рассчитан")
	}
	return *v
}

func TestProgressIsWeighted(t *testing.T) {
	p := Plan{Tasks: []Task{
		{ID: "t1", Weight: 3, Status: TaskCompleted},
		{ID: "t2", Weight: 1, Status: TaskOnReview}, // на проверке — вклада нет
	}}
	if got := progress(t, p); got != 75 {
		t.Fatalf("ожидалось 75, получено %d", got)
	}
}

func TestProgressWithoutTasksIsNotCalculated(t *testing.T) {
	if (Plan{}).Progress() != nil {
		t.Fatal("без заданий процент не должен рассчитываться")
	}
}

func TestProgressFollowsTaskComposition(t *testing.T) {
	p := Plan{Tasks: []Task{{ID: "t1", Weight: 1, Status: TaskCompleted}}}
	if got := progress(t, p); got != 100 {
		t.Fatalf("ожидалось 100, получено %d", got)
	}
	if err := p.AddTask(Task{ID: "t2", Weight: 3}); err != nil {
		t.Fatal(err)
	}
	if got := progress(t, p); got != 25 || p.Tasks[0].Status != TaskCompleted {
		t.Fatalf("после добавления ожидалось 25 и сохранённый статус, получено %d", got)
	}
	if err := p.RemoveTask("t2"); err != nil {
		t.Fatal(err)
	}
	if got := progress(t, p); got != 100 {
		t.Fatalf("после удаления ожидалось 100, получено %d", got)
	}
}

func TestAddTaskRejectsNonPositiveWeight(t *testing.T) {
	var p Plan
	if err := p.AddTask(Task{ID: "t1", Weight: 0}); !errors.Is(err, ErrInvalidWeight) {
		t.Fatalf("ожидался ErrInvalidWeight, получено %v", err)
	}
}

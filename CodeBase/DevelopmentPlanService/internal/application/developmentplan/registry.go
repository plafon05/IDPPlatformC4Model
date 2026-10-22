package developmentplan

import (
	"errors"
	"fmt"
	"sync"

	"idp-platform/development-plan-service/internal/domain/plan"
)

var ErrUnknownCriterion = errors.New("критерий выполнения не поддерживается")

// CompletionStrategy — единый контракт проверки критерия выполнения задания.
// Возвращает true, если результат удовлетворяет критерию задания.
type CompletionStrategy interface {
	Criterion() plan.Criterion
	IsMet(task plan.Task, result TaskResult) (bool, error)
}

// StrategyRegistry хранит стратегии по критерию выполнения.
// Новый тип задания добавляется регистрацией стратегии без изменения сервиса (Open/Closed).
type StrategyRegistry struct {
	mu          sync.RWMutex
	byCriterion map[plan.Criterion]CompletionStrategy
}

func NewStrategyRegistry() *StrategyRegistry {
	return &StrategyRegistry{byCriterion: make(map[plan.Criterion]CompletionStrategy)}
}

// Register добавляет стратегию или заменяет зарегистрированную для того же критерия.
func (r *StrategyRegistry) Register(strategy CompletionStrategy) error {
	if strategy == nil {
		return errors.New("стратегия не задана")
	}
	if strategy.Criterion() == "" {
		return errors.New("у стратегии не задан критерий")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byCriterion[strategy.Criterion()] = strategy
	return nil
}

func (r *StrategyRegistry) Resolve(criterion plan.Criterion) (CompletionStrategy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	strategy, ok := r.byCriterion[criterion]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownCriterion, criterion)
	}
	return strategy, nil
}

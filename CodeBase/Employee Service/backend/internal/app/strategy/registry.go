package strategy

import (
	"fmt"

	"idp-platform/employee-service/internal/domain"
)

// StrategyRegistry - реестр стратегий (Open/Closed Principle)
type StrategyRegistry struct {
	strategies map[domain.TargetService]ChangeDispatchStrategy
}

func NewStrategyRegistry() *StrategyRegistry {
	return &StrategyRegistry{
		strategies: make(map[domain.TargetService]ChangeDispatchStrategy),
	}
}

func (r *StrategyRegistry) Register(strategy ChangeDispatchStrategy) {
	r.strategies[strategy.TargetService()] = strategy
}

func (r *StrategyRegistry) Get(target domain.TargetService) (ChangeDispatchStrategy, error) {
	strat, ok := r.strategies[target]
	if !ok {
		return nil, fmt.Errorf("стратегия для сервиса %s не найдена", target)
	}
	return strat, nil
}


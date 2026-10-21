package delivery

import (
	"context"
	"errors"
	"example.com/idp/delivery-service-l4/internal/domain/deliveryjob"
	"example.com/idp/delivery-service-l4/internal/domain/notification"
	"fmt"
	"sync"
)

type DeliveryStrategy interface {
	Deliver(ctx context.Context, notification notification.Notification) error
}

type StrategyRegistry struct {
	mu        sync.RWMutex
	byChannel map[deliveryjob.Channel]DeliveryStrategy
}

func NewStrategyRegistry() *StrategyRegistry {
	return &StrategyRegistry{byChannel: make(map[deliveryjob.Channel]DeliveryStrategy)}
}

func (r *StrategyRegistry) Register(channel deliveryjob.Channel, strategy DeliveryStrategy) error {
	if r == nil {
		return errors.New("реестр не задан")
	}
	if channel != deliveryjob.ChannelEmail && channel != deliveryjob.ChannelInApp {
		return fmt.Errorf("неизвестный канал: %q", channel)
	}
	if strategy == nil {
		return errors.New("стратегия не задана")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byChannel == nil {
		r.byChannel = make(map[deliveryjob.Channel]DeliveryStrategy)
	}
	r.byChannel[channel] = strategy
	return nil
}

func (r *StrategyRegistry) Resolve(channel deliveryjob.Channel) (DeliveryStrategy, error) {
	if r == nil {
		return nil, errors.New("реестр не задан")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	strategy, ok := r.byChannel[channel]
	if !ok {
		return nil, fmt.Errorf("стратегия для канала %q не зарегистрирована", channel)
	}
	return strategy, nil
}

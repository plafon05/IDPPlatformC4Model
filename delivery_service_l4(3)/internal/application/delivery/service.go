package delivery

import (
	"context"
	"errors"
	"example.com/idp/delivery-service-l4/internal/domain/deliveryjob"
	"example.com/idp/delivery-service-l4/internal/domain/notification"
	"fmt"
)

type DeliveryService struct {
	jobs      DeliveryJobRepository
	registry  *StrategyRegistry
	templates TemplateEngine
}

func NewDeliveryService(jobs DeliveryJobRepository, registry *StrategyRegistry, templates TemplateEngine) (*DeliveryService, error) {
	if jobs == nil || registry == nil || templates == nil {
		return nil, errors.New("не все зависимости DeliveryService заданы")
	}
	return &DeliveryService{jobs: jobs, registry: registry, templates: templates}, nil
}

func (s *DeliveryService) Get(ctx context.Context, id string) (deliveryjob.DeliveryJob, error) {
	if err := ctx.Err(); err != nil {
		return deliveryjob.DeliveryJob{}, err
	}
	if id == "" {
		return deliveryjob.DeliveryJob{}, errors.New("пустой ID задания")
	}
	return s.jobs.Get(ctx, id)
}

func (s *DeliveryService) Send(ctx context.Context, id string) (deliveryjob.DeliveryJob, error) {
	job, err := s.Get(ctx, id)
	if err != nil {
		return job, err
	}
	if err = job.Start(); err != nil {
		return job, err
	}

	strategy, err := s.registry.Resolve(job.Channel)
	if err == nil {
		var rendered notification.Notification
		rendered, err = s.templates.Render(ctx, job.Notification)
		if err == nil {
			err = strategy.Deliver(ctx, rendered)
		}
	}
	if err != nil {
		if stateErr := job.MarkFailed(err.Error()); stateErr != nil {
			return job, errors.Join(err, stateErr)
		}
	} else if stateErr := job.MarkDelivered(); stateErr != nil {
		return job, stateErr
	}

	if saveErr := s.jobs.Save(ctx, job); saveErr != nil {
		return job, fmt.Errorf("не удалось сохранить результат доставки: %w", saveErr)
	}
	// Delivery errors are also returned to the caller after the failed state is saved.
	return job, err
}

package delivery

import (
	"context"
	"errors"
	"example.com/idp/delivery-service-l4/internal/domain/notification"
)

// EmailDeliveryStrategy delegates transport mechanics to EmailSender.
type EmailDeliveryStrategy struct{ sender EmailSender }

func NewEmailDeliveryStrategy(sender EmailSender) (*EmailDeliveryStrategy, error) {
	if sender == nil {
		return nil, errors.New("EmailSender не задан")
	}
	return &EmailDeliveryStrategy{sender: sender}, nil
}
func (s *EmailDeliveryStrategy) Deliver(ctx context.Context, notification notification.Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if notification.EmailAddress == "" {
		return errors.New("не указан email получателя")
	}
	return s.sender.Send(ctx, notification)
}

// InAppDeliveryStrategy saves the notification and then publishes an SSE event.
type InAppDeliveryStrategy struct {
	notifications NotificationRepository
	publisher     Publisher
}

func NewInAppDeliveryStrategy(repo NotificationRepository, publisher Publisher) (*InAppDeliveryStrategy, error) {
	if repo == nil || publisher == nil {
		return nil, errors.New("не заданы зависимости in-app")
	}
	return &InAppDeliveryStrategy{notifications: repo, publisher: publisher}, nil
}
func (s *InAppDeliveryStrategy) Deliver(ctx context.Context, notification notification.Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if notification.UserID == "" {
		return errors.New("не указан пользователь")
	}
	if err := s.notifications.SaveInApp(ctx, notification); err != nil {
		return err
	}
	return s.publisher.Publish(ctx, notification)
}

var _ DeliveryStrategy = (*EmailDeliveryStrategy)(nil)
var _ DeliveryStrategy = (*InAppDeliveryStrategy)(nil)

package delivery

import (
	"context"
	"example.com/idp/delivery-service-l4/internal/domain/deliveryjob"
	"example.com/idp/delivery-service-l4/internal/domain/notification"
)

// The following types are ports only. No adapters are implemented here.
type TemplateEngine interface {
	Render(ctx context.Context, notification notification.Notification) (notification.Notification, error)
}

type DeliveryJobRepository interface {
	Get(ctx context.Context, id string) (deliveryjob.DeliveryJob, error)
	Save(ctx context.Context, job deliveryjob.DeliveryJob) error
}

type NotificationRepository interface {
	SaveInApp(ctx context.Context, notification notification.Notification) error
}

type EmailSender interface {
	Send(ctx context.Context, notification notification.Notification) error
}

type Publisher interface {
	Publish(ctx context.Context, notification notification.Notification) error
}

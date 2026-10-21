package deliveryjob

import (
	"errors"
	"example.com/idp/delivery-service-l4/internal/domain/notification"
	"fmt"
)

type Channel string

const (
	ChannelEmail Channel = "email"
	ChannelInApp Channel = "in_app"
)

type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusDelivered JobStatus = "delivered"
	StatusFailed    JobStatus = "failed"
)

// DeliveryJob owns its state transitions.
type DeliveryJob struct {
	ID           string
	Channel      Channel
	Notification notification.Notification
	Status       JobStatus
	Attempts     int
	LastError    string
}

func (j *DeliveryJob) Start() error {
	if j == nil {
		return errors.New("задание не задано")
	}
	if j.Status != StatusPending {
		return fmt.Errorf("нельзя отправить задание со статусом %q", j.Status)
	}
	if j.Channel != ChannelEmail && j.Channel != ChannelInApp {
		return fmt.Errorf("неизвестный канал: %q", j.Channel)
	}
	if j.Notification.ID == "" {
		return errors.New("уведомление не задано")
	}
	j.Attempts++
	return nil
}

func (j *DeliveryJob) MarkDelivered() error {
	if j == nil || j.Status != StatusPending || j.Attempts == 0 {
		return errors.New("неверный переход в delivered")
	}
	j.Status = StatusDelivered
	j.LastError = ""
	return nil
}

func (j *DeliveryJob) MarkFailed(reason string) error {
	if j == nil || j.Status != StatusPending || j.Attempts == 0 {
		return errors.New("неверный переход в failed")
	}
	if reason == "" {
		return errors.New("не указана причина ошибки")
	}
	j.Status = StatusFailed
	j.LastError = reason
	return nil
}

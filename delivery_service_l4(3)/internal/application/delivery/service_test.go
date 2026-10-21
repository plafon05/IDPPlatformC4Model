package delivery

import (
	"context"
	"errors"
	"example.com/idp/delivery-service-l4/internal/domain/deliveryjob"
	"example.com/idp/delivery-service-l4/internal/domain/notification"
	"testing"
)

// Test doubles are defined only in _test.go. They are not production adapters.
type jobsStub struct {
	items map[string]deliveryjob.DeliveryJob
	saves int
}

func (s *jobsStub) Get(_ context.Context, id string) (deliveryjob.DeliveryJob, error) {
	job, ok := s.items[id]
	if !ok {
		return deliveryjob.DeliveryJob{}, errors.New("not found")
	}
	return job, nil
}
func (s *jobsStub) Save(_ context.Context, job deliveryjob.DeliveryJob) error {
	s.saves++
	s.items[job.ID] = job
	return nil
}

type templatesStub struct{ calls int }

func (s *templatesStub) Render(_ context.Context, n notification.Notification) (notification.Notification, error) {
	s.calls++
	n.Title = "Готово: " + n.Title
	return n, nil
}

type emailStub struct{ calls int }

func (s *emailStub) Send(_ context.Context, _ notification.Notification) error { s.calls++; return nil }

type repoStub struct{ saves int }

func (s *repoStub) SaveInApp(_ context.Context, _ notification.Notification) error {
	s.saves++
	return nil
}

type publisherStub struct{ calls int }

func (s *publisherStub) Publish(_ context.Context, _ notification.Notification) error {
	s.calls++
	return nil
}

type failingStrategy struct{}

func (f failingStrategy) Deliver(context.Context, notification.Notification) error {
	return errors.New("delivery failed")
}

type countingStrategy struct{ calls *int }

func (s countingStrategy) Deliver(context.Context, notification.Notification) error {
	*s.calls++
	return nil
}

func TestEmailDelivery(t *testing.T) {
	ctx := context.Background()
	email := &emailStub{}
	repo := &jobsStub{items: map[string]deliveryjob.DeliveryJob{
		"j1": {ID: "j1", Channel: deliveryjob.ChannelEmail, Status: deliveryjob.StatusPending, Notification: notification.Notification{ID: "n1", EmailAddress: "user@example.org", Title: "Задача"}},
	}}
	registry := NewStrategyRegistry()
	strategy, _ := NewEmailDeliveryStrategy(email)
	_ = registry.Register(deliveryjob.ChannelEmail, strategy)
	template := &templatesStub{}
	svc, _ := NewDeliveryService(repo, registry, template)
	job, err := svc.Send(ctx, "j1")
	if err != nil || job.Status != deliveryjob.StatusDelivered || job.Attempts != 1 || email.calls != 1 || template.calls != 1 || repo.saves != 1 {
		t.Fatalf("неверный результат: job=%+v err=%v", job, err)
	}
	if _, err = svc.Send(ctx, "j1"); err == nil {
		t.Fatal("завершённое задание должно блокировать повтор")
	}
}
func TestInAppDelivery(t *testing.T) {
	repo := &repoStub{}
	pub := &publisherStub{}
	s, _ := NewInAppDeliveryStrategy(repo, pub)
	err := s.Deliver(context.Background(), notification.Notification{ID: "n1", UserID: "u1"})
	if err != nil || repo.saves != 1 || pub.calls != 1 {
		t.Fatalf("save=%d publish=%d error=%v", repo.saves, pub.calls, err)
	}
}
func TestReplaceStrategy(t *testing.T) {
	registry := NewStrategyRegistry()
	first, second := 0, 0
	_ = registry.Register(deliveryjob.ChannelInApp, countingStrategy{calls: &first})
	a, _ := registry.Resolve(deliveryjob.ChannelInApp)
	_ = a.Deliver(context.Background(), notification.Notification{})
	_ = registry.Register(deliveryjob.ChannelInApp, countingStrategy{calls: &second})
	b, _ := registry.Resolve(deliveryjob.ChannelInApp)
	_ = b.Deliver(context.Background(), notification.Notification{})
	if first != 1 || second != 1 {
		t.Fatalf("replacement failed first=%d second=%d", first, second)
	}
}
func TestFailedDeliveryChangesDomainState(t *testing.T) {
	repo := &jobsStub{items: map[string]deliveryjob.DeliveryJob{
		"j2": {ID: "j2", Channel: deliveryjob.ChannelEmail, Status: deliveryjob.StatusPending, Notification: notification.Notification{ID: "n2"}},
	}}
	registry := NewStrategyRegistry()
	_ = registry.Register(deliveryjob.ChannelEmail, failingStrategy{})
	svc, _ := NewDeliveryService(repo, registry, &templatesStub{})
	job, err := svc.Send(context.Background(), "j2")
	if err == nil || job.Status != deliveryjob.StatusFailed || job.Attempts != 1 || job.LastError == "" {
		t.Fatalf("job=%+v err=%v", job, err)
	}
}
func TestMissingStrategy(t *testing.T) {
	repo := &jobsStub{items: map[string]deliveryjob.DeliveryJob{
		"j3": {ID: "j3", Channel: deliveryjob.ChannelEmail, Status: deliveryjob.StatusPending, Notification: notification.Notification{ID: "n3"}},
	}}
	svc, _ := NewDeliveryService(repo, NewStrategyRegistry(), &templatesStub{})
	job, err := svc.Send(context.Background(), "j3")
	if err == nil || job.Status != deliveryjob.StatusFailed {
		t.Fatalf("expected failure: %+v %v", job, err)
	}
}

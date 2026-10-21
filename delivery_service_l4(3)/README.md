# Delivery Service — UML L4 (Go)

Чистая реализация компонента в `internal/application/delivery/`. Внешние взаимодействия — только порты, без реализаций адаптеров.

## Организация

- `internal/domain/notification/notification.go`: агрегат `Notification`.
- `internal/domain/deliveryjob/delivery_job.go`: агрегат `DeliveryJob`, перечисления `Channel` и `JobStatus`, методы перехода состояния `Start`, `MarkDelivered`, `MarkFailed`.
- `internal/application/delivery/ports.go`: `TemplateEngine`, `EmailSender`, `Publisher`, `NotificationRepository`, `DeliveryJobRepository`.
- `internal/application/delivery/registry.go`: `DeliveryStrategy`, `StrategyRegistry` и методы `Register` / `Resolve`.
- `internal/application/delivery/strategies.go`: `EmailDeliveryStrategy` и `InAppDeliveryStrategy`.
- `internal/application/delivery/service.go`: `DeliveryService`, методы `Get` и `Send`.
- `internal/application/delivery/service_test.go`: 5 автоматических тестов с локальными заглушками, не инфраструктурные адаптеры.
- `delivery_service_l4.puml`: UML-диаграмма L4 (расположение исходного файла сохранено).

Запуск проверки из корня проекта:

```bash
go test -v ./...
go vet ./...
```

Нет исполняемого `cmd/demo`, потому что пока нет реализаций портов. Для будущего включения в основной backend потребуется согласовать Go module path с IDP Platform.

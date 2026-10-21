package domain

import "time"

// TargetService - enum целевых сервисов-потребителей платформы IDP
type TargetService string

const (
	TargetEmployee     TargetService = "EMPLOYEE"
	TargetOrganization TargetService = "ORGANIZATION"
	TargetRole         TargetService = "ROLE"
)

// ChangeRecord - чистый доменный объект кадрового изменения (Value Object)
type ChangeRecord struct {
	RecordID      string
	TargetService TargetService
	EntityID      string
	Data          map[string]any
	Timestamp     time.Time
}


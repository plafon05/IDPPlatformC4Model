package notification

// Notification — агрегат уведомления.
type Notification struct {
	ID           string
	UserID       string
	EmailAddress string
	Title        string
	Body         string
}

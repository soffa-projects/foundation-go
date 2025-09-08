package adapters

type NotificationService interface {
	Post(message string) error
}

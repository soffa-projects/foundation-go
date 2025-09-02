package f

type NotificationService interface {
	Post(message string) error
}

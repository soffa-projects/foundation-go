package f

type I18n interface {
	T(messageId string, args ...any) string
}

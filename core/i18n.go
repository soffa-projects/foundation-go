package adapters

import "io/fs"

type I18n interface {
	T(messageId string, args ...any) string
}

type LocalesConfig struct {
	LocaleFS fs.FS
	Locales  string
}

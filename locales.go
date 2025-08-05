package micro

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/thoas/go-funk"
	"golang.org/x/text/language"
)

func createLocalizer(localesFS fs.FS, supportedLocales string) *I18n {
	bundle := i18n.NewBundle(language.French)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	locales := funk.UniqString(strings.Split(supportedLocales, ","))
	for _, lang := range locales {
		localFile := fmt.Sprintf("locales/locale.%s.toml", lang)
		_, err := bundle.LoadMessageFileFS(localesFS, localFile)
		if err != nil {
			LogFatal("unable to load locale file %s", localFile)
			panic(err)
		}
	}
	localizer := i18n.NewLocalizer(bundle, locales...)
	LogInfo("%d locales loaded", len(locales))
	return &I18n{
		localizer: localizer,
	}
}

type I18n struct {
	localizer *i18n.Localizer
}

func (i *I18n) T(messageId string, args ...any) string {
	return i.localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: messageId,
		},
	})
}

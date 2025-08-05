package adapters

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	core "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/log"
	"github.com/thoas/go-funk"
	"golang.org/x/text/language"
)

type i18nImpl struct {
	localizer *i18n.Localizer
}

func NewLocalizer(localesFS fs.FS, supportedLocales string) core.I18n {
	bundle := i18n.NewBundle(language.French)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	locales := funk.UniqString(strings.Split(supportedLocales, ","))
	for _, lang := range locales {
		localFile := fmt.Sprintf("locales/locale.%s.toml", lang)
		_, err := bundle.LoadMessageFileFS(localesFS, localFile)
		if err != nil {
			log.Fatal("unable to load locale file %s", localFile)
			panic(err)
		}
	}
	localizer := i18n.NewLocalizer(bundle, locales...)
	log.Info("%d locales loaded", len(locales))
	return &i18nImpl{
		localizer: localizer,
	}
}

func (i *i18nImpl) T(messageId string, args ...any) string {
	return i.localizer.MustLocalize(&i18n.LocalizeConfig{
		DefaultMessage: &i18n.Message{
			ID: messageId,
		},
	})
}

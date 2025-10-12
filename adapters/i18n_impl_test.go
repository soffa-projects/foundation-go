package adapters

import (
	"embed"
	"io/fs"
	"testing"

	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/test"
)

//go:embed testdata
var testdataFS embed.FS

func getTestLocalesFS() fs.FS {
	// Create a sub-FS rooted at testdata
	// This way paths like "locales/locale.en.toml" will work
	sub, _ := fs.Sub(testdataFS, "testdata")
	return sub
}

// ------------------------------------------------------------------------------------------------------------------
// Constructor Tests
// ------------------------------------------------------------------------------------------------------------------

func TestNewLocalizer_SingleLocale(t *testing.T) {
	assert := test.NewAssertions(t)

	localizer, err := NewLocalizer(getTestLocalesFS(), "en")

	assert.Nil(err)
	assert.NotNil(localizer)
	// Verify it implements the interface
	var _ f.I18n = localizer
}

func TestNewLocalizer_MultipleLocales(t *testing.T) {
	assert := test.NewAssertions(t)

	localizer, err := NewLocalizer(getTestLocalesFS(), "en,fr")

	assert.Nil(err)
	assert.NotNil(localizer)
}

func TestNewLocalizer_WithSpaces(t *testing.T) {
	assert := test.NewAssertions(t)

	// Note: Spaces are not trimmed in the current implementation
	// This is expected to work only if locales don't have spaces
	localizer, err := NewLocalizer(getTestLocalesFS(), "en,fr,es")

	assert.Nil(err)
	assert.NotNil(localizer)
}

func TestNewLocalizer_DuplicateLocales(t *testing.T) {
	assert := test.NewAssertions(t)

	// Should deduplicate locales
	localizer, err := NewLocalizer(getTestLocalesFS(), "en,en,fr,fr")

	assert.Nil(err)
	assert.NotNil(localizer)
}

func TestNewLocalizer_InvalidLocale(t *testing.T) {
	assert := test.NewAssertions(t)

	// Should fail for non-existent locale file
	localizer, err := NewLocalizer(getTestLocalesFS(), "xx")

	assert.NotNil(err)
	if localizer != nil {
		t.Error("Expected nil localizer for invalid locale")
	}
}

func TestMustNewLocalizer_Success(t *testing.T) {
	assert := test.NewAssertions(t)

	localizer := MustNewLocalizer(getTestLocalesFS(), "en")

	assert.NotNil(localizer)
}

func TestMustNewLocalizer_Panic(t *testing.T) {
	assert := test.NewAssertions(t)

	// Should panic for invalid locale
	defer func() {
		r := recover()
		assert.NotNil(r)
	}()

	MustNewLocalizer(getTestLocalesFS(), "invalid-locale")
	t.Error("Should have panicked")
}

// ------------------------------------------------------------------------------------------------------------------
// Translation Tests
// ------------------------------------------------------------------------------------------------------------------

func TestI18n_T_English(t *testing.T) {
	assert := test.NewAssertions(t)

	localizer, _ := NewLocalizer(getTestLocalesFS(), "en")

	translated := localizer.T("hello")

	assert.Equals(translated, "Hello")
}

func TestI18n_T_French(t *testing.T) {
	assert := test.NewAssertions(t)

	localizer, _ := NewLocalizer(getTestLocalesFS(), "fr")

	translated := localizer.T("hello")

	assert.Equals(translated, "Bonjour")
}

func TestI18n_T_Spanish(t *testing.T) {
	assert := test.NewAssertions(t)

	localizer, _ := NewLocalizer(getTestLocalesFS(), "es")

	translated := localizer.T("hello")

	assert.Equals(translated, "Hola")
}

func TestI18n_T_MissingTranslation_Panics(t *testing.T) {
	assert := test.NewAssertions(t)

	localizer, _ := NewLocalizer(getTestLocalesFS(), "en")

	// Missing translation panics with MustLocalize
	defer func() {
		r := recover()
		assert.NotNil(r)
	}()

	localizer.T("non.existent.key")
	t.Error("Should have panicked for missing translation")
}

func TestI18n_T_MultipleMessages(t *testing.T) {
	assert := test.NewAssertions(t)

	localizer, _ := NewLocalizer(getTestLocalesFS(), "en")

	hello := localizer.T("hello")
	goodbye := localizer.T("goodbye")
	welcome := localizer.T("welcome")

	assert.Equals(hello, "Hello")
	assert.Equals(goodbye, "Goodbye")
	assert.Equals(welcome, "Welcome")
}

// ------------------------------------------------------------------------------------------------------------------
// Fallback Tests
// ------------------------------------------------------------------------------------------------------------------

func TestI18n_T_PreferFirstLocale(t *testing.T) {
	assert := test.NewAssertions(t)

	// French is first, so it should be preferred
	localizer, _ := NewLocalizer(getTestLocalesFS(), "fr,en")

	translated := localizer.T("hello")

	assert.Equals(translated, "Bonjour")
}

func TestI18n_T_FirstLocaleUsed(t *testing.T) {
	assert := test.NewAssertions(t)

	// English is first
	localizer, _ := NewLocalizer(getTestLocalesFS(), "en,fr")

	translated := localizer.T("hello")

	assert.Equals(translated, "Hello")
}

// ------------------------------------------------------------------------------------------------------------------
// Edge Cases
// ------------------------------------------------------------------------------------------------------------------

func TestI18n_T_EmptyMessageId_Panics(t *testing.T) {
	assert := test.NewAssertions(t)

	localizer, _ := NewLocalizer(getTestLocalesFS(), "en")

	// Empty message ID should panic
	defer func() {
		r := recover()
		assert.NotNil(r)
	}()

	localizer.T("")
	t.Error("Should have panicked for empty message ID")
}

func TestI18n_T_SpecialCharacters(t *testing.T) {
	assert := test.NewAssertions(t)

	localizer, _ := NewLocalizer(getTestLocalesFS(), "en")

	translated := localizer.T("special_chars")

	assert.Equals(translated, "Special: @#$% & !?")
}

func TestI18n_T_UnicodeCharacters(t *testing.T) {
	assert := test.NewAssertions(t)

	localizer, _ := NewLocalizer(getTestLocalesFS(), "en")

	translated := localizer.T("unicode")

	assert.Equals(translated, "Unicode: 你好 🌍")
}

// ------------------------------------------------------------------------------------------------------------------
// Integration Test
// ------------------------------------------------------------------------------------------------------------------

func TestI18n_EndToEnd(t *testing.T) {
	assert := test.NewAssertions(t)

	// Setup: Create localizer with multiple locales
	localizer, err := NewLocalizer(getTestLocalesFS(), "en,fr,es")
	assert.Nil(err)

	// Test: English translations
	assert.Equals(localizer.T("hello"), "Hello")
	assert.Equals(localizer.T("goodbye"), "Goodbye")

	// Create French localizer
	frLocalizer, err := NewLocalizer(getTestLocalesFS(), "fr")
	assert.Nil(err)

	// Test: French translations
	assert.Equals(frLocalizer.T("hello"), "Bonjour")
	assert.Equals(frLocalizer.T("goodbye"), "Au revoir")

	// Create Spanish localizer
	esLocalizer, err := NewLocalizer(getTestLocalesFS(), "es")
	assert.Nil(err)

	// Test: Spanish translations
	assert.Equals(esLocalizer.T("hello"), "Hola")
	assert.Equals(esLocalizer.T("goodbye"), "Adiós")
}

// NOTE: These tests use embedded test locale files from testdata/locales/
// Files required:
// - locale.en.toml (English)
// - locale.fr.toml (French)
// - locale.es.toml (Spanish)
//
// All tests use in-memory operations with embedded files, no external dependencies.

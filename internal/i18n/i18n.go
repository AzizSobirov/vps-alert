// Package i18n holds the translations of everything vps-alert sends to
// Telegram.
//
// Each area of the code keeps its messages in its own files, one per
// language (bot_en.go, bot_uz.go, bot_ru.go, ...), and registers them from
// init. Keys are "area.name", e.g. "backup.finished".
//
// Messages are Telegram HTML and are always passed through fmt.Sprintf, so:
//   - a literal percent sign is written %%;
//   - every translation must use the same verbs as the English message.
//     When the word order differs, use explicit indexes: %[2]s ... %[1]s.
//   - dynamic values that may contain <, > or & must be escaped by the
//     caller with html.EscapeString before being passed as arguments.
package i18n

import (
	"errors"
	"fmt"
	"strings"
)

type Lang string

const (
	EN Lang = "en"
	UZ Lang = "uz"
	RU Lang = "ru"
)

// Langs lists the supported languages in the order they are offered.
var Langs = []Lang{EN, UZ, RU}

var catalogs = map[Lang]map[string]string{}

func register(l Lang, msgs map[string]string) {
	c := catalogs[l]
	if c == nil {
		c = map[string]string{}
		catalogs[l] = c
	}
	for k, v := range msgs {
		if _, dup := c[k]; dup {
			panic("i18n: duplicate key " + string(l) + ":" + k)
		}
		c[k] = v
	}
}

// T returns the message for key in l, formatted with args. It falls back to
// English, then to the key itself, so a missing translation is visible but
// never fatal.
func T(l Lang, key string, args ...any) string {
	msg, ok := catalogs[l][key]
	if !ok {
		msg, ok = catalogs[EN][key]
	}
	if !ok {
		return key
	}
	return fmt.Sprintf(msg, args...)
}

// Parse accepts a language code such as "uz" or "RU".
func Parse(s string) (Lang, bool) {
	l := Lang(strings.ToLower(strings.TrimSpace(s)))
	for _, x := range Langs {
		if l == x {
			return l, true
		}
	}
	return "", false
}

// Pick returns the first valid language among values, or English. Callers
// pass the bot setting first and the env default second.
func Pick(values ...string) Lang {
	for _, v := range values {
		if l, ok := Parse(v); ok {
			return l
		}
	}
	return EN
}

// Label is the language's name in itself, for buttons.
func (l Lang) Label() string {
	switch l {
	case UZ:
		return "🇺🇿 O'zbekcha"
	case RU:
		return "🇷🇺 Русский"
	default:
		return "🇬🇧 English"
	}
}

const docsBase = "https://github.com/AzizSobirov/vps-alert/blob/main/docs/"

// DocURL links to a documentation page, e.g. "restore.md", in l.
func DocURL(l Lang, page string) string {
	if l == EN {
		return docsBase + page
	}
	return docsBase + string(l) + "/" + page
}

// Error is an error whose message is translated when shown to the user.
type Error struct {
	Key  string
	Args []any
}

// Err returns a translatable error.
func Err(key string, args ...any) error {
	return &Error{Key: key, Args: args}
}

func (e *Error) Error() string { return T(EN, e.Key, e.Args...) }

// Text returns err's message in l if err is or wraps an *Error, and
// err.Error() otherwise (e.g. raw output from restic).
func Text(l Lang, err error) string {
	var e *Error
	if errors.As(err, &e) {
		return T(l, e.Key, e.Args...)
	}
	return err.Error()
}

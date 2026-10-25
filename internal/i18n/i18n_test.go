package i18n

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// verbRe matches fmt verbs, including explicit argument indexes.
var verbRe = regexp.MustCompile(`%(\[\d+\])?[-+# 0]*\d*(\.\d+)?[a-zA-Z%]`)

func verbs(msg string) []string {
	var out []string
	for _, v := range verbRe.FindAllString(msg, -1) {
		if v == "%%" {
			continue
		}
		out = append(out, v[len(v)-1:]) // compare the verb letter only
	}
	sort.Strings(out)
	return out
}

// Every language must have exactly the English keys, with the same verbs.
func TestCatalogsComplete(t *testing.T) {
	en := catalogs[EN]
	if len(en) == 0 {
		t.Skip("no messages registered yet")
	}
	for _, l := range Langs[1:] {
		c := catalogs[l]
		for k, msg := range en {
			tr, ok := c[k]
			if !ok {
				t.Errorf("%s: missing %q", l, k)
				continue
			}
			if a, b := strings.Join(verbs(msg), ","), strings.Join(verbs(tr), ","); a != b {
				t.Errorf("%s: %q uses verbs [%s], English uses [%s]", l, k, b, a)
			}
		}
		for k := range c {
			if _, ok := en[k]; !ok {
				t.Errorf("%s: %q has no English message", l, k)
			}
		}
	}
}

// Formatting every message with placeholder args must not produce
// %!verb(...) errors, which would mean a broken format string.
func TestMessagesFormat(t *testing.T) {
	for _, l := range Langs {
		for k, msg := range catalogs[l] {
			args := make([]any, 0)
			for _, v := range verbRe.FindAllString(msg, -1) {
				switch v[len(v)-1] {
				case '%':
					continue
				case 'd':
					args = append(args, 1)
				case 'f', 'g':
					args = append(args, 1.0)
				default:
					args = append(args, "x")
				}
			}
			if out := fmt.Sprintf(msg, args...); strings.Contains(out, "%!") {
				t.Errorf("%s: %q formats badly: %s", l, k, out)
			}
		}
	}
}

func TestFallbacks(t *testing.T) {
	register("xx", map[string]string{})
	defer delete(catalogs, "xx")
	register(EN, map[string]string{"test.only": "hello %s"})
	defer delete(catalogs[EN], "test.only")

	if got := T("xx", "test.only", "bob"); got != "hello bob" {
		t.Errorf("fallback to English: %q", got)
	}
	if got := T(EN, "test.missing"); got != "test.missing" {
		t.Errorf("missing key: %q", got)
	}
}

func TestPickAndError(t *testing.T) {
	if Pick("", "RU") != RU || Pick("de") != EN || Pick("uz", "ru") != UZ {
		t.Error("Pick")
	}
	register(EN, map[string]string{"test.err": "bad %d"})
	defer delete(catalogs[EN], "test.err")
	err := fmt.Errorf("wrapped: %w", Err("test.err", 3))
	if Text(EN, err) != "bad 3" || Text(EN, errors.New("raw")) != "raw" {
		t.Error("Text")
	}
}

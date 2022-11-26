package http

import (
	"fmt"
	"strings"
	"unicode"
)

func getRouteName(method, path string) string {
	path = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Latin, r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, strings.Trim(path, "/"))
	return strings.ToLower(fmt.Sprintf("%s_%s", method, path))
}

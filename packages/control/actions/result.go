package actions

import (
	"strconv"
	"strings"
)

type Result struct {
	EquivalentCLI string
	Message       string
	Fields        map[string]string
	Items         []map[string]string
	SecretLabel   string
	SecretValue   string
}

func Command(args ...string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, quoteArg(arg))
	}
	return strings.Join(quoted, " ")
}

func quoteArg(arg string) string {
	if arg == "" {
		return strconv.Quote(arg)
	}
	if strings.ContainsAny(arg, " \t\n\r\"'") {
		return strconv.Quote(arg)
	}
	return arg
}

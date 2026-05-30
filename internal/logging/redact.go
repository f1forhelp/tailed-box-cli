package logging

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
)

var (
	sensitiveKey = regexp.MustCompile(`(?i)(join[-_ ]?code|token|secret|password|private[-_ ]?key|authorization|credential|decrypted[-_ ]?payload)`)
	bearerValue  = regexp.MustCompile(`(?i)\bbearer\s+[a-z0-9._~+/=-]+`)
	keyValue     = regexp.MustCompile(`(?i)\b(join[-_ ]?code|token|secret|password|private[-_ ]?key|authorization|credential)\b\s*[:=]\s*["']?[^"',\s}]+`)
)

type RedactingHandler struct {
	Next slog.Handler
}

func (h RedactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.Next.Enabled(ctx, level)
}

func (h RedactingHandler) Handle(ctx context.Context, record slog.Record) error {
	redacted := slog.NewRecord(record.Time, record.Level, Redact(record.Message), record.PC)
	record.Attrs(func(attr slog.Attr) bool {
		redacted.AddAttrs(RedactAttr(attr))
		return true
	})
	return h.Next.Handle(ctx, redacted)
}

func (h RedactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	redacted := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		redacted = append(redacted, RedactAttr(attr))
	}
	return RedactingHandler{Next: h.Next.WithAttrs(redacted)}
}

func (h RedactingHandler) WithGroup(name string) slog.Handler {
	return RedactingHandler{Next: h.Next.WithGroup(name)}
}

func Redact(input string) string {
	if input == "" {
		return input
	}
	input = bearerValue.ReplaceAllString(input, "Bearer <redacted>")
	return keyValue.ReplaceAllStringFunc(input, func(match string) string {
		separator := strings.IndexAny(match, ":=")
		if separator < 0 {
			return "<redacted>"
		}
		return strings.TrimSpace(match[:separator]) + "=<redacted>"
	})
}

func RedactAttr(attr slog.Attr) slog.Attr {
	if sensitiveKey.MatchString(attr.Key) {
		return slog.String(attr.Key, "<redacted>")
	}

	switch attr.Value.Kind() {
	case slog.KindString:
		return slog.String(attr.Key, Redact(attr.Value.String()))
	case slog.KindGroup:
		group := attr.Value.Group()
		redacted := make([]slog.Attr, 0, len(group))
		for _, nested := range group {
			redacted = append(redacted, RedactAttr(nested))
		}
		return slog.Group(attr.Key, attrsToAny(redacted)...)
	default:
		return attr
	}
}

func attrsToAny(attrs []slog.Attr) []any {
	values := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		values = append(values, attr)
	}
	return values
}

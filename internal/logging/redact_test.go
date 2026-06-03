package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	got := Redact("join_code=abc123 token:secret-value Authorization: Bearer abc.def code tbxjc1.abc_DEF.ghi_JKL")
	for _, leaked := range []string{"abc123", "secret-value", "abc.def", "tbxjc1.abc_DEF.ghi_JKL"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("redaction leaked %q in %q", leaked, got)
		}
	}
}

func TestRedactingHandler(t *testing.T) {
	var out bytes.Buffer
	logger := slog.New(RedactingHandler{Next: slog.NewJSONHandler(&out, nil)})
	logger.InfoContext(context.Background(), "enrollment", "join_code", "secret-code", "message", "password=bad")
	got := out.String()
	for _, leaked := range []string{"secret-code", "password=bad"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("handler leaked %q in %s", leaked, got)
		}
	}
}

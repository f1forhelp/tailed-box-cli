package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/f1forhelp/tailed-box-cli/packages/control/actions"
	secureidentity "github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
)

func TestTUICallsControlLayerAndShowsEquivalentCLI(t *testing.T) {
	original := tuiActions
	defer func() { tuiActions = original }()
	called := false
	tuiActions.createJoinCode = func(_ context.Context, role secureidentity.Role, _ ...actions.Option) (actions.Result, error) {
		called = true
		if role != secureidentity.RoleWorker {
			t.Fatalf("role = %q, want worker", role)
		}
		return actions.Result{EquivalentCLI: "infra join-code create --role worker", Message: "ok"}, nil
	}

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), nil, strings.NewReader("6\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d stderr=%s", code, stderr.String())
	}
	if !called {
		t.Fatal("control action was not called")
	}
	output := stdout.String()
	if !strings.Contains(output, "CLI: infra join-code create --role worker") || !strings.Contains(output, "equivalent CLI: infra join-code create --role worker") {
		t.Fatalf("stdout missing equivalent CLI: %s", output)
	}
}

func TestTUIShowsAllPrimaryActions(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), nil, strings.NewReader("q\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d stderr=%s", code, stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"CLI: infra network init",
		"CLI: infra identity init --role master",
		"CLI: infra peer export",
		"CLI: infra mesh ping --endpoint <endpoint>",
		"CLI: infra pair listen --bind 127.0.0.1:9444",
		"CLI: infra pair join --endpoint <endpoint> --code <code> --role worker --master-node <node-id>",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("stdout missing %q: %s", want, output)
		}
	}
}

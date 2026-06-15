package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/f1forhelp/tailed-box-cli/packages/control/actions"
)

func TestCLICallsControlLayer(t *testing.T) {
	original := cliActions
	defer func() { cliActions = original }()
	called := false
	cliActions.initNetwork = func(context.Context, ...actions.Option) (actions.Result, error) {
		called = true
		return actions.Result{EquivalentCLI: "infra network init", Message: "ok"}, nil
	}

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"network", "init"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d stderr=%s", code, stderr.String())
	}
	if !called {
		t.Fatal("control action was not called")
	}
	if !strings.Contains(stdout.String(), "equivalent CLI: infra network init") {
		t.Fatalf("stdout missing equivalent CLI: %s", stdout.String())
	}
}

func TestCLIMeshListenUsesControlLayer(t *testing.T) {
	original := cliActions
	defer func() { cliActions = original }()
	called := false
	cliActions.prepareMesh = func(_ context.Context, bind string, _ ...actions.Option) (actions.MeshListener, error) {
		called = true
		if bind != "127.0.0.1:9443" {
			t.Fatalf("bind = %q", bind)
		}
		return actions.MeshListener{
			EquivalentCLI: "infra mesh listen --bind 127.0.0.1:9443",
			Bind:          bind,
			Addr:          bind,
			Serve: func(context.Context) error {
				return nil
			},
			Close: func() error { return nil },
		}, nil
	}

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"mesh", "listen", "--bind", "127.0.0.1:9443"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d stderr=%s", code, stderr.String())
	}
	if !called {
		t.Fatal("prepare mesh action was not called")
	}
	if !strings.Contains(stdout.String(), "mesh listener started") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestCLIRejectsInvalidCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"unknown"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("invalid command succeeded")
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr = %s", stderr.String())
	}
}

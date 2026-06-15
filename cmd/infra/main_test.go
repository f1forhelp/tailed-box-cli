package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/f1forhelp/tailed-box-cli/packages/control/actions"
	secureidentity "github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
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

func TestCLIPairListenUsesControlLayer(t *testing.T) {
	original := cliActions
	defer func() { cliActions = original }()
	called := false
	cliActions.preparePairing = func(_ context.Context, bind string, _ ...actions.Option) (actions.PairingListener, error) {
		called = true
		if bind != "127.0.0.1:9444" {
			t.Fatalf("bind = %q", bind)
		}
		return actions.PairingListener{
			EquivalentCLI: "infra pair listen --bind 127.0.0.1:9444",
			Bind:          bind,
			Addr:          bind,
			Serve: func(context.Context) error {
				return nil
			},
			Close: func() error { return nil },
		}, nil
	}

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"pair", "listen", "--bind", "127.0.0.1:9444"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d stderr=%s", code, stderr.String())
	}
	if !called {
		t.Fatal("prepare pairing action was not called")
	}
	if !strings.Contains(stdout.String(), "pairing listener started") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestCLIPairJoinUsesControlLayer(t *testing.T) {
	original := cliActions
	defer func() { cliActions = original }()
	called := false
	cliActions.joinPairing = func(_ context.Context, endpoint, code string, role secureidentity.Role, masterNode secureidentity.NodeID, _ ...actions.Option) (actions.Result, error) {
		called = true
		if endpoint != "master.example:9444" || code != "CODE" || role != secureidentity.RoleWorker || masterNode != "node_master" {
			t.Fatalf("unexpected args endpoint=%q code=%q role=%q master=%q", endpoint, code, role, masterNode)
		}
		return actions.Result{EquivalentCLI: "infra pair join --endpoint master.example:9444 --code <code> --role worker --master-node node_master", Message: "ok"}, nil
	}

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"pair", "join", "--endpoint", "master.example:9444", "--code", "CODE", "--role", "worker", "--master-node", "node_master"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d stderr=%s", code, stderr.String())
	}
	if !called {
		t.Fatal("join pairing action was not called")
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

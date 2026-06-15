package actions

import (
	"context"
	"errors"
	"testing"
	"time"

	secureidentity "github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
)

func TestControlActionEquivalentCLI(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	now := func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }
	options := []Option{WithConfigRoot(root), WithClock(now)}

	networkResult, err := InitNetwork(ctx, options...)
	if err != nil {
		t.Fatalf("InitNetwork: %v", err)
	}
	if networkResult.EquivalentCLI != "infra network init" {
		t.Fatalf("network equivalent = %q", networkResult.EquivalentCLI)
	}
	identityResult, err := InitIdentity(ctx, secureidentity.RoleMaster, options...)
	if err != nil {
		t.Fatalf("InitIdentity: %v", err)
	}
	if identityResult.EquivalentCLI != "infra identity init --role master" {
		t.Fatalf("identity equivalent = %q", identityResult.EquivalentCLI)
	}
	joinResult, err := CreateJoinCode(ctx, secureidentity.RoleWorker, options...)
	if err != nil {
		t.Fatalf("CreateJoinCode: %v", err)
	}
	if joinResult.EquivalentCLI != "infra join-code create --role worker" {
		t.Fatalf("join equivalent = %q", joinResult.EquivalentCLI)
	}
	if joinResult.SecretValue == "" {
		t.Fatal("join code secret not returned to caller")
	}
}

func TestWorkerCannotCreateJoinCode(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	options := []Option{WithConfigRoot(root)}
	if _, err := InitNetwork(ctx, options...); err != nil {
		t.Fatalf("InitNetwork: %v", err)
	}
	if _, err := InitIdentity(ctx, secureidentity.RoleWorker, options...); err != nil {
		t.Fatalf("InitIdentity: %v", err)
	}
	_, err := CreateJoinCode(ctx, secureidentity.RoleWorker, options...)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("CreateJoinCode err = %v, want ErrUnauthorized", err)
	}
}

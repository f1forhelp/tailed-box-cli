package actions

import (
	"context"
	"errors"

	secureidentity "github.com/f1forhelp/tailed-box-cli/packages/securemesh/identity"
	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/join"
)

var ErrUnauthorized = errors.New("current node is not authorized for this action")

func CreateJoinCode(ctx context.Context, role secureidentity.Role, options ...Option) (Result, error) {
	if err := checkContext(ctx); err != nil {
		return Result{}, err
	}
	env, err := newEnv(options)
	if err != nil {
		return Result{}, err
	}
	identity, err := secureidentity.LoadIdentity(env.paths)
	if err != nil {
		return Result{}, err
	}
	if identity.Role != secureidentity.RoleMaster {
		return Result{}, ErrUnauthorized
	}

	store := join.NewStore(env.paths).WithClock(env.now)
	code, record, err := store.Create(join.CreateRequest{
		NetworkID:    identity.NetworkID,
		ExpectedRole: role,
		IssuedBy:     identity.NodeID,
	})
	if err != nil {
		return Result{}, err
	}
	return Result{
		EquivalentCLI: Command("infra", "join-code", "create", "--role", role.String()),
		Message:       "join code created",
		Fields: map[string]string{
			"code_id":       record.ID.String(),
			"network_id":    record.NetworkID.String(),
			"expected_role": record.ExpectedRole.String(),
		},
		SecretLabel: "join_code",
		SecretValue: code,
	}, nil
}

func ConsumeJoinCode(ctx context.Context, code string, role secureidentity.Role, options ...Option) (Result, error) {
	if err := checkContext(ctx); err != nil {
		return Result{}, err
	}
	env, err := newEnv(options)
	if err != nil {
		return Result{}, err
	}
	identity, err := secureidentity.LoadIdentity(env.paths)
	if err != nil {
		return Result{}, err
	}

	store := join.NewStore(env.paths).WithClock(env.now)
	result, err := store.ValidateAndConsume(join.ConsumeRequest{
		Code:         code,
		NetworkID:    identity.NetworkID,
		ExpectedRole: role,
		ConsumedBy:   identity.NodeID,
	})
	if err != nil {
		return Result{}, err
	}
	return Result{
		EquivalentCLI: Command("infra", "join-code", "consume", "--code", "<code>", "--role", role.String()),
		Message:       "join code consumed",
		Fields: map[string]string{
			"code_id":  result.Record.ID.String(),
			"consumed": "true",
		},
	}, nil
}

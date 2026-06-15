package actions

import (
	"context"
	"time"

	"github.com/f1forhelp/tailed-box-cli/packages/securemesh/config"
)

type Option func(*Options)

type Options struct {
	ConfigRoot string
	Now        func() time.Time
}

func WithConfigRoot(root string) Option {
	return func(options *Options) {
		options.ConfigRoot = root
	}
}

func WithClock(now func() time.Time) Option {
	return func(options *Options) {
		options.Now = now
	}
}

type env struct {
	paths config.Paths
	now   func() time.Time
}

func newEnv(optionList []Option) (env, error) {
	var options Options
	for _, option := range optionList {
		if option != nil {
			option(&options)
		}
	}

	var (
		paths config.Paths
		err   error
	)
	if options.ConfigRoot == "" {
		paths, err = config.DefaultPaths()
	} else {
		paths, err = config.NewPaths(options.ConfigRoot)
	}
	if err != nil {
		return env{}, err
	}

	now := options.Now
	if now == nil {
		now = time.Now
	}
	return env{paths: paths, now: now}, nil
}

func (e env) nowUTC() time.Time {
	return e.now().UTC()
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

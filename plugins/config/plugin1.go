package config

import (
	"context"
	"time"

	"github.com/roadrunner-server/errors"
)

type Configurer interface {
	GracefulTimeout() time.Duration
	Unmarshal(out any) error
	// UnmarshalKey takes a single key and unmarshal it into a Struct.
	UnmarshalKey(name string, out any) error
	// Has checks if config section exists.
	Has(name string) bool
}

type AllConfig struct {
	RPC struct {
		Listen string `mapstructure:"listen"`
	} `mapstructure:"rpc"`
	Reload struct {
		Enabled  bool     `mapstructure:"enabled"`
		Interval string   `mapstructure:"interval"`
		Patterns []string `mapstructure:"patterns"`
		Services struct {
			HTTP struct {
				Recursive bool     `mapstructure:"recursive"`
				Ignore    []string `mapstructure:"ignore"`
				Patterns  []string `mapstructure:"patterns"`
				Dirs      []string `mapstructure:"dirs"`
			} `mapstructure:"http"`
			Jobs struct {
				Recursive bool     `mapstructure:"recursive"`
				Ignore    []string `mapstructure:"ignore"`
				Dirs      []string `mapstructure:"dirs"`
			} `mapstructure:"jobs"`
			RPC struct {
				Recursive bool     `mapstructure:"recursive"`
				Patterns  []string `mapstructure:"patterns"`
				Dirs      []string `mapstructure:"dirs"`
			} `mapstructure:"rpc"`
		} `mapstructure:"services"`
	} `mapstructure:"reload"`
	Logs struct {
		Mode  string `mapstructure:"mode"`
		Level string `mapstructure:"level"`
	} `mapstructure:"logs"`
}

// ReloadConfig is a Reload configuration point.
type ReloadConfig struct {
	Interval time.Duration
	Patterns []string
	Services map[string]ServiceConfig
}

type ServiceConfig struct {
	Enabled   bool
	Recursive bool
	Patterns  []string
	Dirs      []string
	Ignore    []string
}

type Foo struct {
	configProvider Configurer
}

func (f *Foo) Init(p Configurer) error {
	f.configProvider = p
	return nil
}

func (f *Foo) Serve() chan error {
	const op = errors.Op("foo_plugin_serve")
	errCh := make(chan error, 1)

	r := &ReloadConfig{}
	err := f.configProvider.UnmarshalKey("reload", r)
	if err != nil {
		errCh <- err
	}

	if len(r.Patterns) == 0 {
		errCh <- errors.E(op, errors.Str("should be at least one pattern, but got 0"))
		return errCh
	}

	var allCfg AllConfig
	err = f.configProvider.Unmarshal(&allCfg)
	if err != nil {
		errCh <- errors.E(op, errors.Str("should be at least one pattern, but got 0"))
		return errCh
	}

	if allCfg.RPC.Listen != "tcp://127.0.0.1:6060" {
		errCh <- errors.E(op, errors.Str("RPC.Listen should be parsed"))
		return errCh
	}

	return errCh
}

func (f *Foo) Stop(context.Context) error {
	return nil
}

package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Options struct {
	File      string
	EnvPrefix string
}

type Validator interface {
	Validate() error
}

func Load[T Validator](opts Options, defaults T) (T, error) {
	k := koanf.New(".")

	if err := loadFile(k, opts.File); err != nil {
		return defaults, err
	}

	if opts.EnvPrefix != "" {
		if err := loadEnv(k, opts.EnvPrefix); err != nil {
			return defaults, err
		}
	}

	if err := k.Unmarshal("", &defaults); err != nil {
		return defaults, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := defaults.Validate(); err != nil {
		return defaults, fmt.Errorf("validate config: %w", err)
	}

	return defaults, nil
}

func loadFile(k *koanf.Koanf, cfgFile string) error {
	if cfgFile == "" {
		return nil
	}

	if _, err := os.Stat(cfgFile); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err := k.Load(file.Provider(cfgFile), toml.Parser()); err != nil {
		return fmt.Errorf("load config file: %w", err)
	}

	return nil
}

func loadEnv(k *koanf.Koanf, prefix string) error {
	err := k.Load(env.Provider(prefix, ".", func(s string) string {
		return strings.ReplaceAll(
			strings.ToLower(strings.TrimPrefix(s, prefix)),
			"__", ".",
		)
	}), nil)
	if err != nil {
		return fmt.Errorf("load env: %w", err)
	}

	return nil
}

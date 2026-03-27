package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Validator interface {
	Validate() error
}

type Loader[T Validator] struct {
	file      string
	envPrefix string
	defaults  func() T
}

func NewLoader[T Validator](file string, envPrefix string) *Loader[T] {
	return &Loader[T]{
		file:      file,
		envPrefix: envPrefix,
	}
}

func (l *Loader[T]) WithDefaults(fn func() T) *Loader[T] {
	l.defaults = fn
	return l
}

func (l *Loader[T]) Load() (T, error) {
	var cfg T
	if l.defaults != nil {
		cfg = l.defaults()
	}

	k := koanf.New(".")

	if err := loadFile(k, l.file); err != nil {
		return cfg, err
	}

	if l.envPrefix != "" {
		if err := loadEnv(k, l.envPrefix); err != nil {
			return cfg, err
		}
	}

	if err := k.Unmarshal("", &cfg); err != nil {
		return cfg, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

func (l *Loader[T]) Watch(onChange func(T), logger *slog.Logger) error {
	if l.file == "" {
		return nil
	}
	f := file.Provider(l.file)
	return f.Watch(func(event any, err error) {
		if err != nil {
			logger.Error("watch error", "err", err)
			return
		}

		k := koanf.New(".")
		if err := k.Load(f, toml.Parser()); err != nil {
			logger.Error("reload config failed", "err", err)
			return
		}

		var cfg T
		if err := k.Unmarshal("", &cfg); err != nil {
			logger.Error("unmarshal config failed", "err", err)
			return
		}

		onChange(cfg)
	})
}

func loadFile(k *koanf.Koanf, cfgFile string) error {
	if cfgFile == "" {
		return nil
	}

	if _, err := os.Stat(cfgFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat config file: %w", err)
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

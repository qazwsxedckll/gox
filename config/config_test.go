package config

import (
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testConfig struct {
	Name   string     `koanf:"name"`
	Server testServer `koanf:"server"`
}

func (c testConfig) Validate() error { return nil }

type testServer struct {
	Port int `koanf:"port"`
}

type validatableConfig struct {
	Name string `koanf:"name"`
}

func (c validatableConfig) Validate() error {
	if c.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

func TestLoad(t *testing.T) {
	t.Run("file not found returns zero value", func(t *testing.T) {
		loader := NewLoader[testConfig]("testdata/nonexistent.toml", "")
		cfg, err := loader.Load()
		require.NoError(t, err)
		require.Equal(t, testConfig{}, cfg)
	})

	t.Run("empty file returns zero value", func(t *testing.T) {
		loader := NewLoader[testConfig]("testdata/empty.toml", "")
		cfg, err := loader.Load()
		require.NoError(t, err)
		require.Equal(t, testConfig{}, cfg)
	})

	t.Run("file loads values", func(t *testing.T) {
		loader := NewLoader[testConfig]("testdata/override.toml", "")
		cfg, err := loader.Load()
		require.NoError(t, err)
		require.Equal(t, "from-file", cfg.Name)
		require.Equal(t, 9090, cfg.Server.Port)
	})

	t.Run("env overrides file", func(t *testing.T) {
		t.Setenv("TEST_NAME", "from-env")

		loader := NewLoader[testConfig]("testdata/name_only.toml", "TEST_")
		cfg, err := loader.Load()
		require.NoError(t, err)
		require.Equal(t, "from-env", cfg.Name)
	})

	t.Run("no file with env only", func(t *testing.T) {
		t.Setenv("APP_NAME", "env-only")

		loader := NewLoader[testConfig]("", "APP_")
		cfg, err := loader.Load()
		require.NoError(t, err)
		require.Equal(t, "env-only", cfg.Name)
	})

	t.Run("invalid toml returns error", func(t *testing.T) {
		loader := NewLoader[testConfig]("testdata/invalid.toml", "")
		_, err := loader.Load()
		require.Error(t, err)
	})
}

func TestLoadWithDefaults(t *testing.T) {
	t.Run("defaults used when no file or env", func(t *testing.T) {
		loader := NewLoader[testConfig]("testdata/nonexistent.toml", "").
			WithDefaults(func() testConfig {
				return testConfig{Name: "default-name", Server: testServer{Port: 3000}}
			})
		cfg, err := loader.Load()
		require.NoError(t, err)
		require.Equal(t, "default-name", cfg.Name)
		require.Equal(t, 3000, cfg.Server.Port)
	})

	t.Run("file overrides defaults", func(t *testing.T) {
		loader := NewLoader[testConfig]("testdata/override.toml", "").
			WithDefaults(func() testConfig {
				return testConfig{Name: "default-name", Server: testServer{Port: 3000}}
			})
		cfg, err := loader.Load()
		require.NoError(t, err)
		require.Equal(t, "from-file", cfg.Name)
		require.Equal(t, 9090, cfg.Server.Port)
	})

	t.Run("file partially overrides defaults", func(t *testing.T) {
		loader := NewLoader[testConfig]("testdata/name_only.toml", "").
			WithDefaults(func() testConfig {
				return testConfig{Name: "default-name", Server: testServer{Port: 3000}}
			})
		cfg, err := loader.Load()
		require.NoError(t, err)
		require.Equal(t, "from-file", cfg.Name)
		require.Equal(t, 3000, cfg.Server.Port)
	})
}

func TestLoadWithValidator(t *testing.T) {
	t.Run("validation passes", func(t *testing.T) {
		t.Setenv("V_NAME", "ok")
		loader := NewLoader[validatableConfig]("", "V_")
		cfg, err := loader.Load()
		require.NoError(t, err)
		require.Equal(t, "ok", cfg.Name)
	})

	t.Run("validation fails", func(t *testing.T) {
		loader := NewLoader[validatableConfig]("", "")
		_, err := loader.Load()
		require.Error(t, err)
		require.Contains(t, err.Error(), "name is required")
	})
}

func TestWatch(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	f, err := os.CreateTemp(t.TempDir(), "*.toml")
	require.NoError(t, err)

	_, err = f.WriteString(`name = "initial"`)
	require.NoError(t, err)
	f.Close()

	loader := NewLoader[testConfig](f.Name(), "")
	var got testConfig
	err = loader.Watch(func(cfg testConfig) {
		got = cfg
	}, logger)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(f.Name(), []byte(`name = "updated"`), 0644))

	require.Eventually(t, func() bool {
		return got.Name == "updated"
	}, 1*time.Second, 10*time.Millisecond)
}

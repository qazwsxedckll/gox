package config

import (
	"errors"
	"os"
	"testing"

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
	defaults := testConfig{Name: "default", Server: testServer{Port: 8080}}

	t.Run("file not found uses defaults", func(t *testing.T) {
		cfg, err := Load(Options{File: "nonexistent.toml"}, defaults)
		require.NoError(t, err)
		require.Equal(t, defaults, cfg)
	})

	t.Run("empty file uses defaults", func(t *testing.T) {
		f, err := os.CreateTemp("", "*.toml")
		require.NoError(t, err)
		defer os.Remove(f.Name())
		f.Close()

		cfg, err := Load(Options{File: f.Name()}, defaults)
		require.NoError(t, err)
		require.Equal(t, defaults, cfg)
	})

	t.Run("file overrides defaults", func(t *testing.T) {
		f, err := os.CreateTemp("", "*.toml")
		require.NoError(t, err)
		defer os.Remove(f.Name())

		_, err = f.WriteString(`name = "from-file"
[server]
port = 9090
`)
		require.NoError(t, err)
		f.Close()

		cfg, err := Load(Options{File: f.Name()}, defaults)
		require.NoError(t, err)
		require.Equal(t, "from-file", cfg.Name)
		require.Equal(t, 9090, cfg.Server.Port)
	})

	t.Run("env overrides file", func(t *testing.T) {
		f, err := os.CreateTemp("", "*.toml")
		require.NoError(t, err)
		defer os.Remove(f.Name())

		_, err = f.WriteString(`name = "from-file"`)
		require.NoError(t, err)
		f.Close()

		t.Setenv("TEST_NAME", "from-env")

		cfg, err := Load(Options{File: f.Name(), EnvPrefix: "TEST_"}, defaults)
		require.NoError(t, err)
		require.Equal(t, "from-env", cfg.Name)
	})

	t.Run("no file with env only", func(t *testing.T) {
		t.Setenv("APP_NAME", "env-only")

		cfg, err := Load(Options{EnvPrefix: "APP_"}, defaults)
		require.NoError(t, err)
		require.Equal(t, "env-only", cfg.Name)
	})
}

func TestLoadWithValidator(t *testing.T) {
	t.Run("validation passes", func(t *testing.T) {
		t.Setenv("V_NAME", "ok")
		cfg, err := Load(Options{EnvPrefix: "V_"}, validatableConfig{})
		require.NoError(t, err)
		require.Equal(t, "ok", cfg.Name)
	})

	t.Run("validation fails", func(t *testing.T) {
		_, err := Load(Options{}, validatableConfig{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "name is required")
	})

}

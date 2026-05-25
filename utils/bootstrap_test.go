package utils

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestReadConfGeneratesSecretForNewTemplateConfig(t *testing.T) {
	originalConfigFile := configFile
	originalConfigExampleFile := configExampleFile
	t.Cleanup(func() {
		configFile = originalConfigFile
		configExampleFile = originalConfigExampleFile
		viper.Reset()
	})

	viper.Reset()
	t.Setenv("SECRET", "")

	root := t.TempDir()
	configExampleFile = filepath.Join(root, "config.example.yaml")
	configFile = filepath.Join(root, "config", "config.yaml")

	if err := os.WriteFile(configExampleFile, []byte(`
secret: secret
serve_static: false
server:
  port: 8094
`), 0o644); err != nil {
		t.Fatalf("write config template: %v", err)
	}

	ReadConf()

	secret := viper.GetString("secret")
	if isWeakSecret(secret) {
		t.Fatalf("expected generated strong secret, got %q", secret)
	}
	if len(secret) != 64 {
		t.Fatalf("expected 64-byte generated secret, got %d", len(secret))
	}

	conf := viper.New()
	conf.SetConfigFile(configFile)
	if err := conf.ReadInConfig(); err != nil {
		t.Fatalf("read generated config: %v", err)
	}
	if got := conf.GetString("secret"); got != secret {
		t.Fatalf("expected generated secret to be persisted, got %q want %q", got, secret)
	}
}

func TestGenerateSecretForNewConfigKeepsStrongSecret(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	secret := GenerateChar(64)
	if err := os.WriteFile(path, []byte("secret: "+secret+"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := generateSecretForNewConfig(path); err != nil {
		t.Fatalf("generate config secret: %v", err)
	}

	conf := viper.New()
	conf.SetConfigFile(path)
	if err := conf.ReadInConfig(); err != nil {
		t.Fatalf("read config: %v", err)
	}
	if got := conf.GetString("secret"); got != secret {
		t.Fatalf("expected existing strong secret to be preserved, got %q want %q", got, secret)
	}
}

func TestSaveConfigReplacesExistingConfig(t *testing.T) {
	withTempConfigFiles(t, "secret: "+GenerateChar(64)+"\nchannel: []\n")

	if err := SaveConfig("channel", []string{"gpt-4o"}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	conf := viper.New()
	conf.SetConfigFile(configFile)
	if err := conf.ReadInConfig(); err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	channels := conf.GetStringSlice("channel")
	if len(channels) != 1 || channels[0] != "gpt-4o" {
		t.Fatalf("expected saved channel config, got %#v", channels)
	}
}

func TestSaveConfigKeepsExistingConfigWhenReplaceFails(t *testing.T) {
	originalContent := "secret: " + GenerateChar(64) + "\nchannel: []\n"
	withTempConfigFiles(t, originalContent)

	previousRename := renameConfigFile
	renameConfigFile = func(_, _ string) error {
		return errors.New("simulated rename failure")
	}
	t.Cleanup(func() {
		renameConfigFile = previousRename
	})

	if err := SaveConfig("channel", []string{"gpt-4o"}); err == nil {
		t.Fatalf("expected save config to fail")
	}

	content, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("expected original config to remain readable: %v", err)
	}
	if string(content) != originalContent {
		t.Fatalf("expected original config to remain unchanged, got %q", string(content))
	}

	channels := viper.GetStringSlice("channel")
	if len(channels) != 0 {
		t.Fatalf("expected runtime config to remain unchanged, got %#v", channels)
	}
}

func withTempConfigFiles(t *testing.T, content string) {
	t.Helper()

	originalConfigFile := configFile
	originalConfigTmpFile := configTmpFile
	originalConfigBackupFile := configBackupFile
	t.Cleanup(func() {
		configFile = originalConfigFile
		configTmpFile = originalConfigTmpFile
		configBackupFile = originalConfigBackupFile
		viper.Reset()
	})

	viper.Reset()

	root := t.TempDir()
	configFile = filepath.Join(root, "config", "config.yaml")
	configTmpFile = filepath.Join(root, "config", "config.tmp.yaml")
	configBackupFile = filepath.Join(root, "config", "config.bak.yaml")
	if err := os.MkdirAll(filepath.Dir(configFile), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(configFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		t.Fatalf("read config: %v", err)
	}
}

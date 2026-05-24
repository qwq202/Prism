package utils

import (
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

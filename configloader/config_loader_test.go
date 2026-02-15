package configloader_test

import (
	"os"
	"testing"

	"github.com/GabrielNunesIT/go-libs/configloader"
	"github.com/spf13/pflag"
)

type AppConfig struct {
	Host string `koanf:"host"`
	Port int    `koanf:"port"`
}

func TestConfigLoader_Merge(t *testing.T) {
	// 1. Defaults
	defaults := AppConfig{
		Host: "localhost",
		Port: 8080,
	}

	// 2. File (overrides Host)
	configFile := "config.json"
	configContent := `{"host": "file-host"}`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	defer os.Remove(configFile)

	// 3. Env (overrides Port)
	os.Setenv("APP_PORT", "9090")
	defer os.Unsetenv("APP_PORT")

	loader := configloader.NewConfigLoader(
		configloader.WithDefaults(defaults),
		configloader.WithFile[AppConfig](configFile),
		configloader.WithEnv[AppConfig]("APP_"),
	)

	config, err := loader.Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Host should be from file
	if config.Host != "file-host" {
		t.Errorf("Expected Host to be 'file-host', got '%s'", config.Host)
	}
	// Port should be from env
	if config.Port != 9090 {
		t.Errorf("Expected Port to be 9090, got %d", config.Port)
	}
}

func TestConfigLoader_Flags(t *testing.T) {
	defaults := AppConfig{
		Host: "localhost",
		Port: 8080,
	}

	f := pflag.NewFlagSet("config", pflag.ContinueOnError)
	f.String("host", "default-flag-host", "Host address")
	f.Int("port", 0, "Port number")
	// Simulate parsing flags
	f.Parse([]string{"--host=flag-host", "--port=9091"})

	loader := configloader.NewConfigLoader(
		configloader.WithDefaults(defaults),
		configloader.WithFlags[AppConfig](f),
	)

	config, err := loader.Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config.Host != "flag-host" {
		t.Errorf("Expected Host to be 'flag-host', got '%s'", config.Host)
	}
	if config.Port != 9091 {
		t.Errorf("Expected Port to be 9091, got %d", config.Port)
	}
}

func TestConfigLoader_OrderMatters(t *testing.T) {
	defaults := AppConfig{
		Host: "default-host",
	}

	configFile := "config_order.json"
	configContent := `{"host": "file-host"}`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	defer os.Remove(configFile)

	// Case 1: Defaults first, then File (File should win)
	loader1 := configloader.NewConfigLoader(
		configloader.WithDefaults(defaults),
		configloader.WithFile[AppConfig](configFile),
	)
	config1, _ := loader1.Load()
	if config1.Host != "file-host" {
		t.Errorf("Case 1: Expected 'file-host', got '%s'", config1.Host)
	}

	// Case 2: File first, then Defaults (Defaults should win)
	loader2 := configloader.NewConfigLoader(
		configloader.WithFile[AppConfig](configFile),
		configloader.WithDefaults(defaults),
	)
	config2, _ := loader2.Load()
	if config2.Host != "default-host" {
		t.Errorf("Case 2: Expected 'default-host', got '%s'", config2.Host)
	}
}

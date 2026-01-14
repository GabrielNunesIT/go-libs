// Package configloader provides a generic configuration loader.
package configloader

import (
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
)

// ConfigLoader is a generic configuration loader for type T.
type ConfigLoader[T any] struct {
	k   *koanf.Koanf
	err error
}

// Option is a function that configures the ConfigLoader.
type Option[T any] func(*ConfigLoader[T])

// NewConfigLoader creates a new ConfigLoader for type T.
// It accepts a variable number of Option functions to customize the loader.
func NewConfigLoader[T any](opts ...Option[T]) *ConfigLoader[T] {
	loader := &ConfigLoader[T]{
		k: koanf.New("."),
	}
	for _, opt := range opts {
		opt(loader)
	}
	return loader
}

// Load returns the loaded configuration.
//
//nolint:ireturn // Returns generic type T which might be an interface
func (loader *ConfigLoader[T]) Load() (T, error) {
	var config T
	if loader.err != nil {
		return config, loader.err
	}

	//nolint:wrapcheck // Returning error from external package is intended
	if err := loader.k.Unmarshal("", &config); err != nil {
		return config, err
	}

	return config, nil
}

// WithDefaults sets the default configuration.
func WithDefaults[T any](defaults T) Option[T] {
	return func(loader *ConfigLoader[T]) {
		if loader.err != nil {
			return
		}
		// Load defaults using structs provider.
		// We use "koanf" tag by default, but it can be customized if needed.
		if err := loader.k.Load(structs.Provider(defaults, "koanf"), nil); err != nil {
			loader.err = err
		}
	}
}

// WithFile adds a file source to the loader.
// It automatically detects JSON or YAML based on extension.
func WithFile[T any](path string) Option[T] {
	return func(loader *ConfigLoader[T]) {
		if loader.err != nil {
			return
		}

		var parser koanf.Parser
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".json":
			parser = json.Parser()
		case ".yaml", ".yml":
			parser = yaml.Parser()
		default:
			// Default to JSON or return error?
			// Let's try JSON
			parser = json.Parser()
		}

		if err := loader.k.Load(file.Provider(path), parser); err != nil {
			loader.err = err
		}
	}
}

// WithEnv adds an environment variable source.
// prefix is the environment variable prefix to look for (e.g. "APP_").
func WithEnv[T any](prefix string) Option[T] {
	return func(loader *ConfigLoader[T]) {
		if loader.err != nil {
			return
		}

		// Transform env vars: APP_SERVER_PORT -> server.port
		err := loader.k.Load(env.Provider(prefix, ".", func(s string) string {
			return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(s, prefix)), "_", ".")
		}), nil)

		if err != nil {
			loader.err = err
		}
	}
}

// WithFlags adds a command-line flag source (using standard flag package).
func WithFlags[T any](flags *pflag.FlagSet) Option[T] {
	return func(loader *ConfigLoader[T]) {
		if loader.err != nil {
			return
		}

		if err := loader.k.Load(posflag.Provider(flags, ".", loader.k), nil); err != nil {
			loader.err = err
		}
	}
}

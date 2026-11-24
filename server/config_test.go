package main

import (
	"testing"
	"time"
)

func fakeEnv(m map[string]string) func(string) string {
	return func(key string) string { return m[key] }
}

func TestLoadConfig_Defaults(t *testing.T) {
	cfg, err := loadConfig(fakeEnv(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.port != "8080" || cfg.limit != 5 || cfg.window != 10*time.Second {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadConfig_ReadsEnvironment(t *testing.T) {
	cfg, err := loadConfig(fakeEnv(map[string]string{
		"PORT":   "9090",
		"LIMIT":  "100",
		"WINDOW": "1m",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.port != "9090" || cfg.limit != 100 || cfg.window != time.Minute {
		t.Errorf("config = %+v", cfg)
	}
}

func TestLoadConfig_RejectsBadValues(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{"limit not a number", map[string]string{"LIMIT": "lots"}},
		{"limit zero", map[string]string{"LIMIT": "0"}},
		{"limit negative", map[string]string{"LIMIT": "-5"}},
		{"window not a duration", map[string]string{"WINDOW": "soon"}},
		{"window missing unit", map[string]string{"WINDOW": "10"}},
		{"window zero", map[string]string{"WINDOW": "0s"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := loadConfig(fakeEnv(tc.env)); err == nil {
				t.Errorf("expected an error for %v, got none", tc.env)
			}
		})
	}
}

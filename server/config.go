package main

import (
	"fmt"
	"strconv"
	"time"
)

type config struct {
	port   string
	limit  int
	window time.Duration
}

func loadConfig(getenv func(string) string) (config, error) {

	cfg := config{port: "8080", limit: 5, window: 10 * time.Second}

	if v := getenv("PORT"); v != "" {
		cfg.port = v
	}

	if v := getenv("LIMIT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return config{}, fmt.Errorf("LIMIT must be a positive integer, got %q", v)
		}
		cfg.limit = n
	}

	if v := getenv("WINDOW"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return config{}, fmt.Errorf("WINDOW must be a positive duration like 10s or 1m, got %q", v)
		}
		cfg.window = d
	}

	return cfg, nil
}

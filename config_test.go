package mcacascade

import (
	"strings"
	"testing"
	"time"
)

func TestValidateOK(t *testing.T) {
	if err := testConfig().Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateEmptyTargetsOK(t *testing.T) {
	cfg := testConfig()
	cfg.Targets = nil
	if err := cfg.Validate(); err != nil {
		t.Fatalf("empty targets must be valid (all assignments stay in lobby): %v", err)
	}
}

func TestValidateFailures(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Config)
		wantSub string
	}{
		{"missing base url", func(c *Config) { c.BaseURL = "" }, "base_url"},
		{"bad scheme", func(c *Config) { c.BaseURL = "ftp://arena.example" }, "base_url"},
		{"missing token", func(c *Config) { c.Token = "  " }, "token"},
		{"bad timeout", func(c *Config) { c.Timeout = 0 }, "query_timeout"},
		{"unnormalized target key", func(c *Config) { c.Targets["HOST:1234"] = "x" }, "targets"},
		{"empty backend id", func(c *Config) { c.Targets["1.2.3.4:5"] = " " }, "后端 ID"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testConfig()
			tc.mutate(&cfg)
			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error %q does not mention %q", err, tc.wantSub)
			}
		})
	}
}

func TestDefaultQueryTimeout(t *testing.T) {
	if DefaultQueryTimeout != 2500*time.Millisecond {
		t.Errorf("DefaultQueryTimeout = %v, want 2.5s (aligned with VelocityGate)", DefaultQueryTimeout)
	}
}

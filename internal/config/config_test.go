package config

import (
	"testing"
)

func TestValidateMCPMode(t *testing.T) {
	tests := []struct {
		mode    string
		wantErr bool
	}{
		{"full", false},
		{"router", false},
		{"hybrid", false},
		{"", false}, // empty is valid (defaults to full)
		{"invalid", true},
		{"FULL", true}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.MCP.Mode = tt.mode
			errs := Validate(cfg)

			hasErr := false
			for _, err := range errs {
				if err != nil {
					hasErr = true
					break
				}
			}

			if hasErr != tt.wantErr {
				t.Errorf("Validate(MCP.Mode=%q) hasErr=%v, want %v", tt.mode, hasErr, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfigHasMCPMode(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MCP.Mode != "hybrid" {
		t.Errorf("DefaultConfig().MCP.Mode = %q, want %q", cfg.MCP.Mode, "hybrid")
	}
}

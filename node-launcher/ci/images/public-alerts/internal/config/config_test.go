package config

import (
	"testing"
)

func TestValidateChainLagMonitorConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  ChainLagMonitorConfig
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: ChainLagMonitorConfig{
				MaxChainLag: map[string]int{"BTC": 1, "ETH": 1},
			},
			wantErr: false,
		},
		{
			name: "invalid configuration with zero value",
			config: ChainLagMonitorConfig{
				MaxChainLag: map[string]int{"BTC": 0, "ETH": 1},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

package monitor

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	openapi "gitlab.com/thorchain/thornode/openapi/gen"
)

// testDataFetcher with predefined responses for each invariant
type testDataFetcher struct {
	TestData map[string]*openapi.InvariantResponse
}

func (tdf *testDataFetcher) FetchInvariantData(invariant string) (*openapi.InvariantResponse, error) {
	data, exists := tdf.TestData[invariant]
	if !exists {
		return nil, fmt.Errorf("invariant %s not found in test data", invariant)
	}
	return data, nil
}

// setupTestDataFetcher initializes the testDataFetcher with example responses
func setupTestDataFetcher() *testDataFetcher {
	return &testDataFetcher{
		TestData: map[string]*openapi.InvariantResponse{
			"asgard": {
				Invariant: "asgard",
				Broken:    false,
				Msg:       []string{"Asgard is secure."},
			},
			"bond": {
				Invariant: "bond",
				Broken:    true,
				Msg:       []string{"Bond requirement not met."},
			},
			"thorchain": {
				Invariant: "thorchain",
				Broken:    false,
				Msg:       []string{"Thorchain operating within parameters."},
			},
			"affiliate_collector": {
				Invariant: "affiliate_collector",
				Broken:    true,
				Msg:       []string{"Affiliate collector discrepancies detected."},
			},
			"pools": {
				Invariant: "pools",
				Broken:    false,
				Msg:       []string{"All pools balanced."},
			},
			"streaming_swaps": {
				Invariant: "streaming_swaps",
				Broken:    true,
				Msg:       []string{"Streaming swap rates error."},
			},
		},
	}
}

func TestInvariantsMonitor_CheckInvariants(t *testing.T) {
	tests := []struct {
		name         string
		invariant    string
		wantBroken   bool
		wantErrorMsg string
	}{
		{"Test Asgard Invariant", "asgard", false, ""},
		{"Test Bond Invariant", "bond", true, ""},
		{"Test Thorchain Invariant", "thorchain", false, ""},
		{"Test Affiliate Collector Invariant", "affiliate_collector", true, ""},
		{"Test Pools Invariant", "pools", false, ""},
		{"Test Streaming Swaps Invariant", "streaming_swaps", true, ""},
		{"Test Non-Existent Invariant", "nonexistent", false, "invariant nonexistent not found in test data"},
	}

	// TestCheckInvariants tests the CheckInvariants function using testDataFetcher
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDF := setupTestDataFetcher()
			invm := NewInvariantsMonitor()
			invCheck, err := invm.CheckInvariants([]string{tt.invariant}, testDF)
			if err != nil {
				if !strings.Contains(err.Error(), tt.wantErrorMsg) {
					t.Errorf("CheckInvariants() error = %v, wantErr containing %s", err, tt.wantErrorMsg)
				}
			} else {
				if tt.wantBroken {
					assert.Contains(t, invCheck, tt.invariant)
				}
				if !tt.wantBroken {
					assert.NotContains(t, invCheck, tt.invariant)
				}
			}
		},
		)
	}
}

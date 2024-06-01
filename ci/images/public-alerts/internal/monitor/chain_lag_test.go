package monitor

import (
	"testing"

	openapi "gitlab.com/thorchain/thornode/openapi/gen"
)

// TestCalculateChainLag tests the calculateChainLag function for various scenarios.
func TestCalculateChainLag(t *testing.T) {
	// Define test cases
	tests := []struct {
		name              string
		nodes             []openapi.Node
		maxChainLag       map[string]int
		expectedMsgs      []string
		expectedLagCounts map[string]int
	}{
		{
			name: "Single node no lag",
			nodes: []openapi.Node{
				{
					Status: "Active",
					ObserveChains: []openapi.ChainHeight{
						{Chain: "BTC", Height: 100},
					},
				},
			},
			maxChainLag:       map[string]int{"BTC": 10},
			expectedMsgs:      []string{},
			expectedLagCounts: map[string]int{"BTC": 0},
		},
		{
			name: "Multiple nodes with significant lag",
			nodes: []openapi.Node{
				{
					Status: "Active",
					ObserveChains: []openapi.ChainHeight{
						{Chain: "BTC", Height: 100},
						{Chain: "ETH", Height: 300},
					},
				},
				{
					Status: "Active",
					ObserveChains: []openapi.ChainHeight{
						{Chain: "BTC", Height: 80},
						{Chain: "ETH", Height: 250},
					},
				},
			},
			maxChainLag: map[string]int{"BTC": 15, "ETH": 40},
			expectedMsgs: []string{
				"[BTC] Lagging by over 15 blocks on 1 nodes.",
				"[ETH] Lagging by over 40 blocks on 1 nodes.",
			},
			expectedLagCounts: map[string]int{"BTC": 1, "ETH": 1},
		},
	}

	// Execute test cases
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			msgs, lagCounts := calculateChainLag(test.nodes, test.maxChainLag)

			// Check messages
			if len(msgs) != len(test.expectedMsgs) {
				t.Errorf("Expected %d messages, got %d", len(test.expectedMsgs), len(msgs))
			}

			for i, msg := range msgs {
				if msg != test.expectedMsgs[i] {
					t.Errorf("Expected message '%s', got '%s'", test.expectedMsgs[i], msg)
				}
			}

			// Check lag counts
			for chain, count := range test.expectedLagCounts {
				if lagCounts[chain] != count {
					t.Errorf("Expected lag count for %s: %d, got %d", chain, count, lagCounts[chain])
				}
			}
		})
	}
}

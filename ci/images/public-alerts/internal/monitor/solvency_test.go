package monitor

import (
	"public-alerts/internal/config"
	"strings"
	"testing"
)

func TestCheckSolvency(t *testing.T) {

	cfg := config.Config{
		SolvencyMonitor: config.SolvencyMonitorConfig{
			AlertPercentThreshold: 0.05,
			AlertUSDThreshold:     1000,
			AlertWindowThreshold:  60,
			AlertCooldownSeconds:  60 * 60 * 12,
		},
	}

	vaults := []Vault{{
		Addresses: []Address{{
			Chain:   "BTC",
			Address: "1BitcoinAddress",
		}},
		Coins: []Coin{{
			ChainAmount: "900",  // What is Actually in the vault
			Amount:      "1000", // What TC thinks is in the vault
			Asset:       "BTC",
		}},
		Status: "ActiveVault",
		PubKey: "pubKey1",
		Type:   "Hot",
	}}

	assetPrices := map[string]float64{
		"BTC": 50000,
	}
	// check for insolvency condition, chain and vault are different
	alerts, err := checkSolvency(cfg, vaults, assetPrices)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedMsg := "Insolvency detected for BTC at 1Bit...ress"
	if len(alerts) != 1 || !strings.Contains(alerts[0].Message, expectedMsg) {
		t.Errorf("Expected message to contain '%s', got '%s'", expectedMsg, alerts[0].Message)
	}

	// Test for no insolvency condition, chain and vault are the same
	vaults[0].Coins[0].Amount = "900"
	alerts, err = checkSolvency(cfg, vaults, assetPrices)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(alerts) != 0 {
		t.Errorf("Expected no alerts, got %d", len(alerts))
	}
}

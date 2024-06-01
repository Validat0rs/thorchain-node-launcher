package monitor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"public-alerts/internal/common"
	"public-alerts/internal/config"
	"public-alerts/internal/notify"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type Address struct {
	Chain   string `json:"chain"`
	Address string `json:"address"`
}

type Coin struct {
	ChainAmount string `json:"chain_amount"`
	Amount      string `json:"amount"`
	Asset       string `json:"asset"`
}

type Vault struct {
	Status    string    `json:"status"`
	Addresses []Address `json:"addresses"`
	Coins     []Coin    `json:"coins"`
	PubKey    string    `json:"pub_key"`
	Type      string    `json:"type"`
}

type Insolvency struct {
	Asset     string
	Address   string
	Vault     string
	Type      string
	ThorChain string
	Actual    string
	Diff      string
	USD       float64
}

type SolvencyMonitor struct {
}

func (solvm *SolvencyMonitor) Name() string {
	return "SolvencyMonitor"
}

func (solvm *SolvencyMonitor) Check() ([]notify.Alert, error) {

	log.Info().Msg("Checking Solvency...")
	cfg := config.Get()
	vaults, err := fetchSolvencyData(cfg.Endpoints.NineRealmsAPI)
	if err != nil {
		return nil, err
	}

	assetPrices, err := common.AssetToUSDViaMidgard(cfg.Endpoints.MidgardAPI)
	if err != nil {
		return nil, err
	}

	return checkSolvency(cfg, vaults, assetPrices)
}

////////////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////////////

func fetchSolvencyData(apiURL string) ([]Vault, error) {
	resp, err := http.Get(fmt.Sprintf("%s/thorchain/solvency/asgard", apiURL))
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	var vaults []Vault
	if err := json.NewDecoder(resp.Body).Decode(&vaults); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}
	return vaults, nil
}

func FindAddress(addresses []Address, asset string) string {

	// Iterate through the list of addresses
	for _, address := range addresses {
		// Check if the current address's chain matches the asset
		asset_chain := strings.Split(asset, ".")[0]

		if address.Chain == asset_chain {
			return address.Address
		}
	}
	// Return an empty string or a default address if no match is found
	return ""
}

////////////////////////////////////////////////////////////////////////////////
// Check Solvency
////////////////////////////////////////////////////////////////////////////////

func checkSolvency(cfg config.Config, vaults []Vault, assetPrices map[string]float64) ([]notify.Alert, error) {
	var insolvencies []Insolvency

	for _, vault := range vaults {
		if vault.Status != "ActiveVault" {
			continue
		}

		for _, coin := range vault.Coins {
			chainAmount, err := strconv.Atoi(coin.ChainAmount)
			if err != nil || chainAmount == 0 {
				continue
			}

			amount, err := strconv.Atoi(coin.Amount)
			if err != nil {
				continue
			}

			diff := chainAmount - amount
			pctDiff := float64(diff) / float64(chainAmount)

			// calculate USD diff
			assetPrice, ok := assetPrices[coin.Asset]
			if !ok {
				continue
			}

			usdDiff := float64(diff) * assetPrice

			// TODO: verify this condition with Ursa
			// if pctDiff is negative and usdDiff is less than the % threshold
			// AND the USDDiff (should be negative too) is less than the USDThreshold then alert.

			// Check insolvency conditions
			if float64(pctDiff) <= -cfg.SolvencyMonitor.AlertPercentThreshold && usdDiff < cfg.SolvencyMonitor.AlertUSDThreshold {
				insolvencies = append(insolvencies, Insolvency{
					Asset:     coin.Asset,
					Address:   common.ShortenAddress(FindAddress(vault.Addresses, coin.Asset)),
					Vault:     common.ShortenPubKey(vault.PubKey),
					Type:      vault.Type,
					ThorChain: strconv.Itoa(amount),
					Actual:    strconv.Itoa(chainAmount),
					Diff:      common.FormatPercent(pctDiff),
					USD:       usdDiff,
				})
			}
		}
	}

	// Compose alerts based on insolvencies
	var alertMsgs []string

	if len(insolvencies) > 0 {
		for _, insolvency := range insolvencies {

			msg := fmt.Sprintf("Insolvency detected for %s at %s", insolvency.Asset, insolvency.Address)
			alertMsgs = append(alertMsgs, msg)
		}
		finalMsg := "```" + strings.Join(alertMsgs, "\n") + "```"

		return []notify.Alert{{Webhooks: cfg.Webhooks.Activity, Message: finalMsg}}, nil
	} else {
		return nil, nil
	}

}

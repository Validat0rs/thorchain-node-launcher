package monitor

import (
	"encoding/json"
	"fmt"
	"net/http"

	"public-alerts/internal/common"
	"public-alerts/internal/config"
	"public-alerts/internal/notify"

	"github.com/rs/zerolog/log"
	openapi "gitlab.com/thorchain/thornode/openapi/gen"
)

// OutboundMonitor monitors transactions that are stuck in outbound processes.
type OutboundMonitor struct {
	seen map[string]bool
}

func NewOutboundMonitor() *OutboundMonitor {
	return &OutboundMonitor{
		seen: make(map[string]bool),
	}
}

func (om *OutboundMonitor) Name() string {
	return "StuckOutboundMonitor"
}

// Check fetches and evaluates outbound transactions to determine if they are stuck.
func (om *OutboundMonitor) Check() ([]notify.Alert, error) {
	log.Info().Msg("Checking for stuck outbound txs...")

	client, err := common.NewThornodeClient()
	if err != nil {
		log.Err(err).Msg("error creating thornode client")
		return nil, err
	}

	currentHeight, err := client.GetLatestHeight()

	if err != nil {
		log.Err(err).Msg("error fetching current height")

		return nil, err
	}

	outbounds, err := getOutboundTransactions()
	if err != nil {
		return nil, err
	}

	var alerts []notify.Alert

	for _, outbound := range outbounds {
		if _, seen := om.seen[*outbound.InHash]; !seen {
			// get txDetails
			txDetails, err := getTxDetails(outbound.InHash)
			if err != nil {
				// log the error and continue to the next transaction
				log.Error().Err(err).Msgf("error fetching transaction details for: %s", *outbound.InHash)
				continue
			}

			// Check if FinalisedHeight is not nil before deref
			if txDetails.FinalisedHeight != nil {
				finalisedHeight := int(*txDetails.FinalisedHeight)
				age := currentHeight - finalisedHeight

				if age > config.Get().StuckOutboundMonitor.BlockAgeThreshold {
					alertMsg := fmt.Sprintf("Stuck transaction detected: %s (%s %s)",
						fmt.Sprintf("%s/tx/%s", config.Get().Endpoints.ExplorerURL, *outbound.InHash), outbound.Coin.Amount, outbound.Coin.Asset)
					alerts = append(alerts, notify.Alert{Message: alertMsg})
					om.seen[*outbound.InHash] = true
				}
			} else {
				// Handle the case where FinalisedHeight is nil
				log.Error().Msgf("FinalisedHeight is nil, cannot calculate age. InHash: %s", *outbound.InHash)
			}

		}
	}

	return alerts, nil
}

// getOutboundTransactions fetches outbound transactions from the THORNode API.
func getOutboundTransactions() ([]openapi.TxOutItem, error) {

	resp, err := http.Get(fmt.Sprintf("%s/thorchain/queue/outbound", config.Get().Endpoints.ThornodeAPI))
	if err != nil {
		return nil, fmt.Errorf("error fetching outbound transactions: %w", err)
	}
	defer resp.Body.Close()

	var transactions []openapi.TxOutItem

	if err := json.NewDecoder(resp.Body).Decode(&transactions); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}
	return transactions, nil
}

// getTxDetails fetches transaction details from the THORNode API using the transaction hash.
func getTxDetails(inHash *string) (openapi.TxDetailsResponse, error) {
	// Define a variable to hold the response.
	var txDetails openapi.TxDetailsResponse

	// Fetch the data from the API.
	resp, err := http.Get(fmt.Sprintf("%s/thorchain/tx/details/%s", config.Get().Endpoints.ThornodeAPI, *inHash))
	if err != nil {
		return txDetails, fmt.Errorf("error fetching transaction details: %w", err)
	}
	defer resp.Body.Close()

	// Decode the JSON response into txDetails.
	if err := json.NewDecoder(resp.Body).Decode(&txDetails); err != nil {
		return txDetails, fmt.Errorf("error decoding JSON: %w", err)
	}

	// Return the populated txDetails struct.
	return txDetails, nil
}

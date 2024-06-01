package monitor

import (
	"fmt"
	"public-alerts/internal/common"
	"public-alerts/internal/config"
	"public-alerts/internal/notify"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	openapi "gitlab.com/thorchain/thornode/openapi/gen"
)

var (
	lastChainLag = make(map[string]int)
	lastAlert    = time.Now()
)

type ChainLagMonitor struct {
}

func (clm *ChainLagMonitor) Name() string {
	return "ChainLagMonitor"
}

////////////////////////////////////////////////////////////////////////////////
// Helpers
////////////////////////////////////////////////////////////////////////////////

func max(slice []int) int {
	max := slice[0]
	for _, v := range slice {
		if v > max {
			max = v
		}
	}
	return max
}

////////////////////////////////////////////////////////////////////////////////
// Calculate Chain Lag
////////////////////////////////////////////////////////////////////////////////

func calculateChainLag(nodes []openapi.Node, maxChainLag map[string]int) ([]string, map[string]int) {
	chainHeights := make(map[string][]int)
	activeNodes := 0
	for _, node := range nodes {
		if node.Status != "Active" {
			continue
		}
		for _, c := range node.ObserveChains {
			chainHeights[c.Chain] = append(chainHeights[c.Chain], int(c.GetHeight()))
		}
		activeNodes++
	}

	var msgs []string
	newLagCounts := make(map[string]int)
	for chain, heights := range chainHeights {
		maxLag, ok := maxChainLag[chain]
		if !ok {
			continue
		}

		maxHeight := max(heights)
		lagCount := 0
		for _, h := range heights {
			if maxHeight-h > maxLag {
				lagCount++
			}
		}

		if lagCount > activeNodes/4 {

			msgs = append(msgs, fmt.Sprintf("[%s] Lagging by over %d blocks on %d nodes.", chain, maxLag, lagCount))
			log.Warn().
				Str("chain", chain).
				Int("maxLag", maxLag).
				Int("lagCount", lagCount).
				Msgf("Lagging by over %d blocks on %d nodes.", maxLag, lagCount)
			newLagCounts[chain]++
		} else {
			newLagCounts[chain] = 0
		}
	}
	return msgs, newLagCounts
}

////////////////////////////////////////////////////////////////////////////////
// Check
////////////////////////////////////////////////////////////////////////////////

func (clm *ChainLagMonitor) Check() ([]notify.Alert, error) {

	log.Info().Msg("Checking Chain Lag...")
	cfg := config.Get()
	client, err := common.NewThornodeClient()
	if err != nil {
		return nil, err
	}

	nodes, err := client.GetNodes()
	if err != nil {
		return nil, err
	}

	msgs, newLagCounts := calculateChainLag(nodes, cfg.ChainLagMonitor.MaxChainLag)

	// Update global state
	for chain, count := range newLagCounts {
		lastChainLag[chain] = count
	}

	if len(msgs) > 0 && time.Since(lastAlert) > time.Hour {
		msg := "```" + fmt.Sprintln(strings.Join(msgs, "\n")) + "```"
		lastAlert = time.Now()

		alerts := []notify.Alert{
			{Webhooks: cfg.Webhooks.Activity, Message: msg},
		}
		return alerts, nil
	}
	return nil, nil
}

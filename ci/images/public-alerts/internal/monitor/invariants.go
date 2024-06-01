package monitor

import (
	"fmt"
	"strings"

	"public-alerts/internal/common"
	"public-alerts/internal/notify"

	"github.com/rs/zerolog/log"
	openapi "gitlab.com/thorchain/thornode/openapi/gen"
)

////////////////////////////////////////////////////////////////////////////////
// DataFetcher
////////////////////////////////////////////////////////////////////////////////

// DataFetcher is an interface for fetching data for invariant checks.
type DataFetcher interface {
	FetchInvariantData(invariant string) (*openapi.InvariantResponse, error)
}

// liveDataFetcher implements the DataFetcher interface for live data.
type liveDataFetcher struct {
	client common.ThornodeDataFetcher
}

func (ldf *liveDataFetcher) FetchInvariantData(invariant string) (*openapi.InvariantResponse, error) {
	return ldf.client.GetInvariant(invariant)
}

func NewLiveDataFetcher() *liveDataFetcher {
	client, err := common.NewThornodeClient()
	if err != nil {
		log.Error().Err(err).Msg("error creating thornode client")
		return nil
	}
	return &liveDataFetcher{
		client: client,
	}
}

////////////////////////////////////////////////////////////////////////////////
// Monitor
////////////////////////////////////////////////////////////////////////////////

type InvariantsMonitor struct {
	tripped map[string]bool // Map to track which invariants have been previously detected as broken.
}

func NewInvariantsMonitor() *InvariantsMonitor {
	return &InvariantsMonitor{
		tripped: make(map[string]bool),
	}
}
func (invm *InvariantsMonitor) Name() string {
	return "InvariantsMonitor"
}

func (invm *InvariantsMonitor) Check() ([]notify.Alert, error) {
	log.Info().Msg("Checking invariants...")

	ldf := NewLiveDataFetcher()
	if ldf == nil {
		return nil, fmt.Errorf("error creating live data fetcher")
	}
	invariants, err := ldf.client.GetInvariants()

	if err != nil {
		return nil, err
	}

	broken, err := invm.CheckInvariants(invariants, ldf)
	if err != nil {
		return nil, err
	}

	log.Info().Msg(fmt.Sprintf("%d new broken invariants", len(broken)))
	if len(broken) > 0 {
		msgs := []string{"> ### Broken Invariants:"}
		for _, b := range broken {

			msgs = append(msgs, fmt.Sprintf("> https://thornode.ninerealms.com/thorchain/invariant/%s", b))
		}
		// Notify using the configured notification system.
		return []notify.Alert{{Message: strings.Join(msgs, "\n")}}, nil
	}

	return nil, nil
}

////////////////////////////////////////////////////////////////////////////////
// CheckInvariants
////////////////////////////////////////////////////////////////////////////////

func (inv *InvariantsMonitor) CheckInvariants(invariants []string, df DataFetcher) ([]string, error) {
	broken := make([]string, 0)

	for _, invariant := range invariants {
		if invariant == "asgard" || invariant == "pools" {
			continue
		}

		invData, err := df.FetchInvariantData(invariant)
		if err != nil {
			log.Error().Err(err).Msgf("error getting invariant: %s", invariant)
			return nil, err
		}

		if invData.Broken && !inv.tripped[invData.Invariant] {
			broken = append(broken, invData.Invariant)
			inv.tripped[invData.Invariant] = true
		} else if !invData.Broken {
			delete(inv.tripped, invData.Invariant)
		}
	}

	return broken, nil
}

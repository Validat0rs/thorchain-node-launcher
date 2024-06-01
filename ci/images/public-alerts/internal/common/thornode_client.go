package common

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"public-alerts/internal/config"
	"time"

	tmhttp "github.com/tendermint/tendermint/rpc/client/http"
	openapi "gitlab.com/thorchain/thornode/openapi/gen"
)

// ThornodeDataFetcher defines the interface for fetching data from Thornode API and RPC.
type ThornodeDataFetcher interface {
	GetLatestHeight() (int, error)
	GetNodes() ([]openapi.Node, error)
	GetInvariants() ([]string, error)
	GetInvariant(invariant string) (*openapi.InvariantResponse, error)
}

// thornodeClient implements the ThornodeDataFetcher interface using Thornode's HTTP and RPC endpoints.
type thornodeClient struct {
	httpClient *http.Client
	rpcClient  *tmhttp.HTTP // Tendermint RPC client
	baseURL    string
}

// NewThornodeClient creates a new client for interacting with Thornode.
func NewThornodeClient() (ThornodeDataFetcher, error) {
	rpcClient, err := tmhttp.New(config.Get().Endpoints.ThornodeRPC, "/websocket")
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC client: %w", err)
	}
	return &thornodeClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		rpcClient:  rpcClient,
		baseURL:    config.Get().Endpoints.ThornodeAPI,
	}, nil
}

// GetLatestHeight returns the latest block height from the Thornode network.
func (c *thornodeClient) GetLatestHeight() (int, error) {
	ctx := context.Background()
	status, err := c.rpcClient.Status(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get current height: %w", err)
	}
	return int(status.SyncInfo.LatestBlockHeight), nil
}

// GetNodes retrieves the list of nodes from the Thornode network.
func (c *thornodeClient) GetNodes() ([]openapi.Node, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/thorchain/nodes", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	var nodes []openapi.Node
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}
	return nodes, nil
}

// GetInvariants retrieves a list of invariants from the Thornode network.
func (c *thornodeClient) GetInvariants() ([]string, error) {

	type InvariantsResp struct {
		Invariants []string `json:"invariants"`
	}
	var invars InvariantsResp
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/thorchain/invariants", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&invars); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}

	return invars.Invariants, nil
}

// GetInvariant returns the status of a specific invariant from the Thornode network.
func (c *thornodeClient) GetInvariant(invariant string) (*openapi.InvariantResponse, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/thorchain/invariant/%s", c.baseURL, invariant))
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	var response openapi.InvariantResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}
	return &response, nil
}

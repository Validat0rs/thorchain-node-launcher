package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type PriceCache struct {
	sync.Mutex

	lastUpdated time.Time
	data        map[string]float64
}

var priceCache = &PriceCache{
	data: make(map[string]float64),
}

func ShortenAddress(address string) string {
	if len(address) > 10 {
		return address[:4] + "..." + address[len(address)-4:]
	}
	return address
}

func ShortenPubKey(pubKey string) string {
	if len(pubKey) > 10 {
		return pubKey[len(pubKey)-4:]
	}
	return pubKey
}

func FormatPercent(value float64) string {
	return fmt.Sprintf("%.2f%%", value*100)
}

// assetToUSDViaMidgard fetches asset prices from the Midgard API and caches them.
// TODO: update to use thornode prices after thorchain/thornode!3478
func AssetToUSDViaMidgard(midgardAPI string) (map[string]float64, error) {
	priceCache.Lock()
	// Check if cache is valid
	if time.Since(priceCache.lastUpdated) < 2*time.Minute {
		defer priceCache.Unlock()
		return priceCache.data, nil
	}
	priceCache.Unlock()

	// Update cache
	priceCache.Lock()
	defer priceCache.Unlock()

	// Double-checking in case another goroutine has already updated the cache
	// prob dont need at this point?
	if time.Since(priceCache.lastUpdated) < 2*time.Minute {
		return priceCache.data, nil
	}

	// Fetch new data
	resp, err := http.Get(fmt.Sprintf("%s/v2/pools", midgardAPI))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get pools: status code %d", resp.StatusCode)
	}

	var pools []struct {
		Asset         string `json:"asset"`
		AssetPriceUSD string `json:"assetPriceUSD"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&pools); err != nil {
		return nil, err
	}

	newCache := make(map[string]float64)
	for _, pool := range pools {
		price, err := strconv.ParseFloat(pool.AssetPriceUSD, 64)
		if err != nil {
			continue // or handle error as you prefer
		}
		newCache[pool.Asset] = price
	}

	// Update the cache with new data
	priceCache.data = newCache
	priceCache.lastUpdated = time.Now()

	return priceCache.data, nil
}

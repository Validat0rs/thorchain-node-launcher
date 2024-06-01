package monitor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"public-alerts/internal/config"
	"public-alerts/internal/notify"
	"sync"

	"github.com/rs/zerolog/log"
)

var detected = make(map[string]int)
var mu sync.Mutex

type ChainUpdateMonitor struct {
	Daemons map[string]config.DaemonConfig
}

func (cup *ChainUpdateMonitor) Name() string {
	return "ChainUpdateMonitor"
}

func NewChainUpdateMonitor() *ChainUpdateMonitor {

	daemons := config.Get().ChainUpdateMonitor.Daemons

	return &ChainUpdateMonitor{daemons}
}

////////////////////////////////////////////////////////////////////////////////
// helpers
////////////////////////////////////////////////////////////////////////////////

func fetchReleases(daemonInfo config.DaemonConfig) ([]struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}, error) {

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases", daemonInfo.Github)

	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		log.Err(err).Msgf("Failed to fetch releases: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var releases []struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		log.Err(err).Msgf("Failed to decode response: %v", err)
		return nil, err
	}
	return releases, nil
}

func fetchLatestSeenTags(daemonInfo config.DaemonConfig) (config.DaemonConfig, error) {

	path := filepath.Join(config.Get().ChainUpdateMonitor.DataDir, daemonInfo.Name)
	if _, err := os.Stat(path); err == nil {
		if tag, err := os.ReadFile(path); err == nil {
			daemonInfo.LatestTag = string(tag)

		}
	} else {
		daemonInfo.LatestTag = ""
	}
	return daemonInfo, nil
}

func writeLatestTag(daemonInfo config.DaemonConfig) (notify.Alert, error) {
	path := filepath.Join(config.Get().ChainUpdateMonitor.DataDir, daemonInfo.Name)
	if err := os.WriteFile(path, []byte(daemonInfo.LatestTag), 0644); err != nil {
		err_msg := fmt.Sprintf("Failed to update latest tag for %s: %v", daemonInfo.Name, err)
		log.Err(err).Msg(err_msg)
		return notify.Alert{Webhooks: config.Get().Webhooks.Errors, Message: err_msg}, err
	}
	log.Info().Msgf("[writeLatestTag] Updated latest tag for %s: %s", daemonInfo.Name, daemonInfo.LatestTag)
	return notify.Alert{}, nil
}

////////////////////////////////////////////////////////////////////////////////
// checkChainUpdates
////////////////////////////////////////////////////////////////////////////////

func checkChainUpdates(daemonInfo config.DaemonConfig) ([]notify.Alert, error) {

	var internalAlert []notify.Alert

	daemonReleases, err := fetchReleases(daemonInfo)

	if err != nil {
		err_msg := fmt.Sprintf("Failed to decode response for %s: %v", daemonInfo.Name, err)
		log.Err(err).Msg(err_msg)
		err_alert := notify.Alert{Webhooks: config.Get().Webhooks.Errors, Message: err_msg}
		internalAlert = append(internalAlert, err_alert)
		return internalAlert, err
	}

	if len(daemonReleases) == 0 {
		log.Warn().Msgf("No releases found for %s", daemonInfo.Name)

		alert := notify.Alert{Webhooks: config.Get().Webhooks.Errors, Message: fmt.Sprintf("No releases found for %s", daemonInfo.Name)}
		internalAlert = append(internalAlert, alert)
		return internalAlert, nil
	}

	log.Info().Msgf("Found %d releases for %s", len(daemonReleases), daemonInfo.Name)
	if len(daemonReleases) > 0 {

		latest := daemonReleases[0].TagName
		if daemonInfo.LatestTag == "" {
			// deal with case where the latest tag is empty (first run)
			log.Info().Msgf("No latest tag found for %s", daemonInfo.Name)
			daemonInfo.LatestTag = latest
			err_alert, err := writeLatestTag(daemonInfo)
			if err != nil {
				internalAlert = append(internalAlert, err_alert)
				return internalAlert, err
			}

		}
		if latest != daemonInfo.LatestTag {

			if detected[daemonInfo.Name] < 3 { //check 3 times before sending a discord message
				mu.Lock()
				defer mu.Unlock()

				detected[daemonInfo.Name]++

				return internalAlert, nil
			} else { // detected more than 3 times, fire alert
				mu.Lock()
				defer mu.Unlock()
				detected[daemonInfo.Name] = 0

				log.Info().Msg("prepping to update latest tag")
				message := fmt.Sprintf("%s Update: current=%s latest=%s %s", daemonInfo.Name, daemonInfo.LatestTag, latest, daemonReleases[0].HTMLURL)
				internalAlert = append(internalAlert, notify.Alert{Webhooks: config.Get().Webhooks.Activity, Message: message})
				daemonInfo.LatestTag = latest // update
				err_alert, err := writeLatestTag(daemonInfo)
				if err != nil {
					internalAlert = append(internalAlert, err_alert)
					return internalAlert, err
				}
			}
		}

	} else {
		mu.Lock()
		defer mu.Unlock()
		detected[daemonInfo.Name] = 0

		return internalAlert, nil

	}
	return internalAlert, nil
}

////////////////////////////////////////////////////////////////////////////////
// Check
////////////////////////////////////////////////////////////////////////////////

func (cup *ChainUpdateMonitor) Check() ([]notify.Alert, error) {
	log.Info().Msg("Checking for chain updates...")
	var allAlerts []notify.Alert

	for _, daemonInfo := range cup.Daemons {

		daemonInfo, err := fetchLatestSeenTags(daemonInfo)

		if err != nil {
			err_msg := fmt.Sprintf("Failed to fetch latest seen tags for %s: %v", daemonInfo.Name, err)
			log.Err(err).Msg(err_msg)
			err_alert := notify.Alert{Webhooks: config.Get().Webhooks.Errors, Message: err_msg}
			allAlerts = append(allAlerts, err_alert)
			continue
		}
		if daemonInfo.Github != "" {
			daemonAlert, err := checkChainUpdates(daemonInfo)
			if err != nil {
				return daemonAlert, err
			} else {
				allAlerts = append(allAlerts, daemonAlert...)
			}
		}
	}
	return allAlerts, nil
}

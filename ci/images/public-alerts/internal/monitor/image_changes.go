package monitor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"public-alerts/internal/config"
	"public-alerts/internal/notify"
	"regexp"

	"github.com/rs/zerolog/log"
)

type ImageChangeMonitor struct {
}

func (img *ImageChangeMonitor) Name() string {
	return "ImageChangeMonitor"
}

func NewImageChangeMonitor() *ImageChangeMonitor {

	return &ImageChangeMonitor{}
}

type Image struct {
	Repo         string `json:"repo"`
	Tag          string `json:"tag"`
	Hash         string `json:"hash"`
	PreviousHash string `json:"previous_hash"`
}

// keep track of images that have been observed
var (
	seen         = make(map[string]string)
	IMAGE_FILTER = regexp.MustCompile(`^thorchain/((devops/node-launcher.*)|(thornode:(chaosnet-multichain|mainnet)-\d+\.\d+\.\d+)|(midgard:\d+\.\d+\.\d+))$`)
)

// //////////////////////////////////////////////////////////////////////////////
// helpers
// //////////////////////////////////////////////////////////////////////////////

func FetchImages() ([]Image, error) {
	//TODO - switch to new non-9R API endpoint, when available
	response, err := http.Get(fmt.Sprintf("%s/thorchain/security/images", config.Get().Endpoints.NineRealmsAPI))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch images, status code: %d", response.StatusCode)
	}

	var images []Image
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&images); err != nil {
		return nil, err
	}

	return images, nil
}

////////////////////////////////////////////////////////////////////////////////
// checkImageChanges
////////////////////////////////////////////////////////////////////////////////

func checkImageChanges(fetchFunc func() ([]Image, error)) ([]notify.Alert, error) {

	// get images
	images, err := fetchFunc()
	if err != nil {
		err_msg := fmt.Sprintf("Failed to fetch images: %v", err)
		return []notify.Alert{{Webhooks: config.Get().Webhooks.Errors, Message: err_msg}}, err
	}

	var newImageTags []string
	var modifiedImageTags = map[string]string{}
	// init the check when first run
	if len(seen) == 0 {

		for _, image := range images {
			key := fmt.Sprintf("%s:%s", image.Repo, image.Tag)
			seen[key] = image.Hash

		}

	} else {
		for _, image := range images {

			if image.Hash == "" {
				log.Warn().Msgf("Image hash not found for %s:%s", image.Repo, image.Tag)
				continue
			}
			imageTag := fmt.Sprintf("%s:%s", image.Repo, image.Tag)
			if !IMAGE_FILTER.MatchString(imageTag) {
				continue
			}
			// Record new and changed for alert message
			if _, found := seen[imageTag]; !found {
				newImageTags = append(newImageTags, imageTag)
			} else if seen[imageTag] != image.Hash {
				modifiedImageTags[imageTag] = seen[imageTag]
			}

			// Update seen
			seen[imageTag] = image.Hash
		}
	}
	log.Debug().Msgf("Seen images after processing: %v", seen)

	// Alerting
	// Prepare and log modified thornode image messages for security
	var securityAlerts []notify.Alert
	pattern := regexp.MustCompile(`thornode`)
	for imageTag, oldHash := range modifiedImageTags {
		if pattern.MatchString(imageTag) {
			msg := fmt.Sprintf("Modified image tag: `%s`\n\tOld: `%s`\n\tNew: `%s`", imageTag, oldHash, seen[imageTag])
			alert := notify.Alert{Webhooks: config.Get().Webhooks.Security, Message: msg}
			securityAlerts = append(securityAlerts, alert)
		}
	}

	// Prepare and log modified and new messages for mainnet-info
	var mainnetInfoAlerts []notify.Alert
	for imageTag, oldHash := range modifiedImageTags {
		msg := fmt.Sprintf("Modified image tag: `%s`\n\tOld: `%s`\n\tNew: `%s`", imageTag, oldHash, seen[imageTag])
		modifiedAlert := notify.Alert{Webhooks: config.Get().Webhooks.Activity, Message: msg}
		mainnetInfoAlerts = append(mainnetInfoAlerts, modifiedAlert)
	}
	for _, imageTag := range newImageTags {
		msg := fmt.Sprintf("New image tag: `%s`", imageTag)
		newImageAlert := notify.Alert{Webhooks: config.Get().Webhooks.Activity, Message: msg}
		mainnetInfoAlerts = append(mainnetInfoAlerts, newImageAlert)
	}
	// Combine alerts and return
	allAlerts := append(mainnetInfoAlerts, securityAlerts...)
	return allAlerts, nil
}

// //////////////////////////////////////////////////////////////////////////////
// Check
// //////////////////////////////////////////////////////////////////////////////
func (img *ImageChangeMonitor) Check() ([]notify.Alert, error) {
	log.Info().Msg("Checking for image changes...")
	log.Debug().Msgf("Seen images: %v", seen)

	alerts, err := checkImageChanges(FetchImages)
	if err != nil {
		err_msg := fmt.Sprintf("Failed to check for image changes: %v", err)
		return []notify.Alert{{Webhooks: config.Get().Webhooks.Activity, Message: err_msg}}, err
	}
	return alerts, nil

}

package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"public-alerts/internal/config"
	"sync"
)

type Alert struct {
	Webhooks config.Webhooks
	Message  string
}

func Notify(alert Alert) []error {
	var wg sync.WaitGroup
	// TODO: not sure about buffer size
	errChan := make(chan error, 10)
	// handle concurrent notifications
	notifyConcurrently := func(webhook string, payload map[string]string) {
		defer wg.Done()
		if err := notify(payload, webhook); err != nil {
			errChan <- err
		}
	}

	// Start goroutines for each webhook
	if alert.Webhooks.Slack != "" {
		wg.Add(1)
		go notifyConcurrently(alert.Webhooks.Slack, map[string]string{"text": alert.Message})
	}
	if alert.Webhooks.Discord != "" {
		wg.Add(1)
		go notifyConcurrently(alert.Webhooks.Discord, map[string]string{"content": alert.Message})
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(errChan) // Close the channel to signal completion

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	return errs
}

func notify(payload map[string]string, webhook string) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling message to JSON: %v", err)
	}

	resp, err := http.Post(webhook, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("error posting message to Discord: %v", err)
	}
	defer resp.Body.Close()

	// Check for both http.StatusOK and http.StatusNoContent
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("received non-success HTTP status: %s", resp.Status)
	}
	return nil
}

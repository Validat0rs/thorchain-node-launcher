package monitor

import (
	"fmt"
	"public-alerts/internal/config"
	"public-alerts/internal/notify"
	"time"

	"github.com/rs/zerolog/log"
)

type Monitor interface {
	Check() ([]notify.Alert, error)
	Name() string
}

// Spawn starts a monitor in a goroutine
func Spawn(m Monitor, alertQueue chan<- notify.Alert, interval time.Duration) {
	go func() {
		// Create a ticker that sends ticks at the specified poll frequency
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// avoid swallowing panic
		defer func() {
			if rec := recover(); rec != nil {
				err_msg := fmt.Sprintf("```[ERROR] public-alerts: Monitor %s panicked: %v ```", m.Name(), rec)
				alertQueue <- notify.Alert{Webhooks: config.Get().Webhooks.Errors, Message: err_msg}
				log.Fatal().Msg(err_msg)
			}
		}()

		// Run the monitor Check at each tick
		for range ticker.C {

			alerts, err := m.Check()

			if err != nil {
				err_msg := fmt.Sprintf("```[ERROR] public-alerts: Error Running monitor %s: %v```", m.Name(), err)
				log.Error().Err(err).Msg(err_msg)
				err_alert := notify.Alert{Webhooks: config.Get().Webhooks.Errors, Message: err_msg}
				alertQueue <- err_alert
			}

			for _, alert := range alerts {
				alertQueue <- alert
			}
		}
	}()
}

package subscriptions

import (
	"context"
	"time"

	"github.com/radamesvaz/bakery-app/internal/logger"
)

func RunWorker(ctx context.Context, svc *Service, intervalHours int) {
	if intervalHours <= 0 {
		intervalHours = 24
	}

	ticker := time.NewTicker(time.Duration(intervalHours) * time.Hour)
	defer ticker.Stop()

	logger.Info().
		Int("interval_hours", intervalHours).
		Int("grace_days", svc.GraceDays).
		Msg("Subscription worker: started")

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Subscription worker: stopping")
			return
		case <-ticker.C:
			logger.Info().Msg("Subscription worker: starting run")
			result, err := svc.ProcessTransitions(ctx, time.Now().UTC())
			if err != nil {
				logger.Err(err).Msg("Subscription worker: run failed")
				continue
			}
			logger.Info().
				Int64("to_pending", result.MarkedPending).
				Int64("to_canceled", result.MarkedCanceled).
				Msg("Subscription worker: run finished")
		}
	}
}

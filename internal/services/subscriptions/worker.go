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
			runAt := time.Now().UTC()
			start := time.Now()
			logger.Info().Time("run_at", runAt).Msg("Subscription worker: starting run")
			result, err := svc.ProcessTransitions(ctx, runAt)
			durationMs := time.Since(start).Milliseconds()
			if err != nil {
				logger.Err(err).
					Time("run_at", runAt).
					Int64("duration_ms", durationMs).
					Int("error_count", 1).
					Msg("Subscription worker: run failed")
				continue
			}
			logger.Info().
				Time("run_at", runAt).
				Int64("duration_ms", durationMs).
				Int("error_count", 0).
				Int64("to_pending_count", result.MarkedPending).
				Int64("to_canceled_count", result.MarkedCanceled).
				Msg("Subscription worker: run finished")
		}
	}
}

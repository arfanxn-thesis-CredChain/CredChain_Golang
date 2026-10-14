package jobs

import (
	"context"
	"fmt"

	"CredChain_Golang/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// MigrateRiver runs River's own table migrations (idempotent).
// Called from `make migrate-up` alongside the Postgres migrations.
func MigrateRiver(ctx context.Context, cfg *config.Config) error {
	pool, err := pgxpool.New(ctx, *cfg.PostgresDSN)
	if err != nil {
		return err
	}
	defer pool.Close()
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return err
	}
	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	return err
}

// RiverEnqueuer implements the Enqueuer interface by inserting River jobs.
type RiverEnqueuer struct {
	client *river.Client[pgx.Tx]
	cfg    *config.Config
}

func (e *RiverEnqueuer) EnqueueExtract(ctx context.Context, args CredentialExtractArgs) error {
	_, err := e.client.Insert(ctx, args, &river.InsertOpts{
		MaxAttempts: *e.cfg.CredentialExtractWorkerMaxAttempts,
	})
	return err
}

func (e *RiverEnqueuer) EnqueueCredentialNotifications(ctx context.Context, args []CredentialNotificationArgs) error {
	if len(args) == 0 {
		return nil
	}
	params := make([]river.InsertManyParams, len(args))
	for i, a := range args {
		params[i] = river.InsertManyParams{
			Args: a,
			InsertOpts: &river.InsertOpts{
				MaxAttempts: 5,
				UniqueOpts:  river.UniqueOpts{ByArgs: true},
			},
		}
	}
	_, err := e.client.InsertMany(ctx, params)
	return err
}

// NewRiverClient creates a pgx pool + River client with the extraction and notification workers,
// and registers lifecycle start/stop. Returns the client and an Enqueuer.
func NewRiverClient(
	lc fx.Lifecycle,
	cfg *config.Config,
	extractWorker *CredentialExtractWorker,
	notificationWorker *CredentialNotificationWorker,
	logger *zap.Logger,
) (Enqueuer, error) {
	pool, err := pgxpool.New(context.Background(), *cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}
	workers := river.NewWorkers()
	river.AddWorker(workers, extractWorker)
	river.AddWorker(workers, notificationWorker)
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:       map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: *cfg.RiverMaxWorkers}},
		Workers:      workers,
		ErrorHandler: extractWorker,
	})
	if err != nil {
		pool.Close()
		return nil, err
	}

	// runCtx is the long-lived context River uses for its notifier loop and
	// worker pool. It MUST NOT be derived from the FX OnStart context, which
	// has a start-phase deadline (default 15s). If River's notifier inherits
	// that deadline, it spins on context.DeadlineExceeded after expiry —
	// listenAndWait fails instantly, CancellableSleep returns instantly,
	// and the reconnect loop floods stderr with millions of identical errors
	// per second. See River notifier.go: only context.Canceled breaks the
	// loop, not DeadlineExceeded.
	runCtx, cancelRun := context.WithCancel(context.Background())
	started := false
	lc.Append(fx.Hook{
		// river.Client.Start is non-blocking: it launches the worker pool in
		// the background and returns immediately. Call it synchronously so the
		// "started" flag is set before OnStop can run (no Start/Stop race).
		OnStart: func(_ context.Context) error {
			if startErr := client.Start(runCtx); startErr != nil {
				cancelRun()
				pool.Close()
				return fmt.Errorf("start river client: %w", startErr)
			}
			started = true
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if started {
				if stopErr := client.Stop(ctx); stopErr != nil {
					logger.Error("failed to stop river client", zap.Error(stopErr))
				}
			}
			cancelRun()
			pool.Close()
			return nil
		},
	})
	return &RiverEnqueuer{client: client, cfg: cfg}, nil
}

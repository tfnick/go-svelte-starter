package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/tfnick/go-svelte-starter/api/db"
	"github.com/tfnick/go-svelte-starter/api/framework/logging"
	"maragu.dev/goqite"
	"maragu.dev/goqite/jobs"
)

const (
	QueueScheduledTasks      = "scheduled-tasks"
	QueueDomainEvents        = "domain-events"
	QueueIntegrationWebhooks = "integration-webhooks"
	QueueHeavyTasks          = "heavy-tasks"
)

type Manager struct {
	queues map[string]*goqite.Queue
	log    *zerolog.Logger
}

type JobFunc func(context.Context, []byte) error

type Runner struct {
	runner *jobs.Runner
}

type JSONRunner struct {
	queue        *goqite.Queue
	limit        int
	pollInterval time.Duration
	extend       time.Duration
	log          *zerolog.Logger
	handler      JobFunc
}

type SendOptions struct {
	Queue    string
	Body     []byte
	Delay    time.Duration
	Priority int
}

func NewManager() (*Manager, error) {
	sqlDB, err := db.SQLDBFor("app")
	if err != nil {
		return nil, fmt.Errorf("get app sql db for queue manager: %w", err)
	}

	logger := logging.For("queue")
	return &Manager{
		queues: map[string]*goqite.Queue{
			QueueScheduledTasks: goqite.New(goqite.NewOpts{
				DB:        sqlDB,
				Name:      QueueScheduledTasks,
				SQLFlavor: goqite.SQLFlavorSQLite,
			}),
			QueueDomainEvents: goqite.New(goqite.NewOpts{
				DB:        sqlDB,
				Name:      QueueDomainEvents,
				SQLFlavor: goqite.SQLFlavorSQLite,
			}),
			QueueIntegrationWebhooks: goqite.New(goqite.NewOpts{
				DB:        sqlDB,
				Name:      QueueIntegrationWebhooks,
				SQLFlavor: goqite.SQLFlavorSQLite,
			}),
			QueueHeavyTasks: goqite.New(goqite.NewOpts{
				DB:        sqlDB,
				Name:      QueueHeavyTasks,
				SQLFlavor: goqite.SQLFlavorSQLite,
			}),
		},
		log: &logger,
	}, nil
}

func (m *Manager) Send(ctx context.Context, opts SendOptions) (string, error) {
	q, err := m.queue(opts.Queue)
	if err != nil {
		return "", err
	}

	message := goqite.Message{
		Body:     opts.Body,
		Delay:    opts.Delay,
		Priority: opts.Priority,
	}

	if tx, ok := db.SQLTxFor(ctx, "app"); ok {
		id, err := q.SendAndGetIDTx(ctx, tx, message)
		return string(id), err
	}

	id, err := q.SendAndGetID(ctx, message)
	return string(id), err
}

func (m *Manager) SendJSON(ctx context.Context, opts SendOptions, payload any) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	opts.Body = body
	return m.Send(ctx, opts)
}

func (m *Manager) NewRunner(queueName string, limit int, pollInterval time.Duration) (*Runner, error) {
	q, err := m.queue(queueName)
	if err != nil {
		return nil, err
	}

	runner := jobs.NewRunner(jobs.NewRunnerOpts{
		Limit:        limit,
		Log:          queueLogger{log: m.log},
		PollInterval: pollInterval,
		Queue:        q,
	})
	return &Runner{runner: runner}, nil
}

func (m *Manager) NewJSONRunner(queueName string, limit int, pollInterval time.Duration) (*JSONRunner, error) {
	q, err := m.queue(queueName)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 1
	}
	if pollInterval <= 0 {
		pollInterval = 100 * time.Millisecond
	}

	return &JSONRunner{
		queue:        q,
		limit:        limit,
		pollInterval: pollInterval,
		extend:       5 * time.Second,
		log:          m.log,
	}, nil
}

func (r *Runner) Register(name string, job JobFunc) {
	r.runner.Register(name, func(ctx context.Context, message []byte) error {
		return job(ctx, message)
	})
}

func (r *Runner) Start(ctx context.Context) {
	r.runner.Start(ctx)
}

func (r *JSONRunner) Register(job JobFunc) {
	r.handler = job
}

func (r *JSONRunner) Start(ctx context.Context) {
	if r.handler == nil {
		r.log.Info().Msg("json queue runner has no handler")
		return
	}

	r.log.Info().Int("limit", r.limit).Msg("starting json queue runner")
	var wg sync.WaitGroup
	sem := make(chan struct{}, r.limit)

	for {
		select {
		case <-ctx.Done():
			r.log.Info().Msg("stopping json queue runner")
			wg.Wait()
			r.log.Info().Msg("stopped json queue runner")
			return
		case sem <- struct{}{}:
		}

		message, err := r.queue.ReceiveAndWait(ctx, r.pollInterval)
		if err != nil {
			<-sem
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			r.log.Info().Err(err).Msg("error receiving json queue message")
			time.Sleep(time.Second)
			continue
		}
		if message == nil {
			<-sem
			continue
		}

		wg.Add(1)
		go func(message *goqite.Message) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if rec := recover(); rec != nil {
					r.log.Info().Str("id", string(message.ID)).Interface("error", rec).Msg("recovered from panic in json queue message")
				}
			}()

			jobCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			go r.extendWhileRunning(jobCtx, message.ID)

			r.log.Info().Str("id", string(message.ID)).Msg("running json queue message")
			before := time.Now()
			if err := r.handler(jobCtx, message.Body); err != nil {
				r.log.Info().Str("id", string(message.ID)).Err(err).Msg("json queue message failed")
				return
			}
			r.log.Info().Str("id", string(message.ID)).Dur("duration", time.Since(before)).Msg("ran json queue message")

			deleteCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
			defer cancel()
			if err := r.queue.Delete(deleteCtx, message.ID); err != nil {
				r.log.Info().Str("id", string(message.ID)).Err(err).Msg("error deleting json queue message")
			}
		}(message)
	}
}

func (r *JSONRunner) extendWhileRunning(ctx context.Context, id goqite.ID) {
	delay := r.extend - r.extend/5
	if delay <= 0 {
		delay = time.Second
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if err := r.queue.Extend(ctx, id, r.extend); err != nil {
				r.log.Info().Str("id", string(id)).Err(err).Msg("error extending json queue message timeout")
			}
			timer.Reset(delay)
		}
	}
}

func (m *Manager) CreateJob(ctx context.Context, queueName string, name string, body []byte, delay time.Duration, priority int) (string, error) {
	q, err := m.queue(queueName)
	if err != nil {
		return "", err
	}

	message := goqite.Message{
		Body:     body,
		Delay:    delay,
		Priority: priority,
	}

	if tx, ok := db.SQLTxFor(ctx, "app"); ok {
		id, err := jobs.CreateTx(ctx, tx, q, name, message)
		return string(id), err
	}

	id, err := jobs.Create(ctx, q, name, message)
	return string(id), err
}

func (m *Manager) queue(name string) (*goqite.Queue, error) {
	q := m.queues[name]
	if q == nil {
		return nil, fmt.Errorf("queue not registered: %s", name)
	}
	return q, nil
}

type queueLogger struct {
	log *zerolog.Logger
}

func (l queueLogger) Info(msg string, args ...any) {
	event := l.log.Info()
	for i := 0; i+1 < len(args); i += 2 {
		key, ok := args[i].(string)
		if !ok {
			continue
		}
		event = event.Interface(key, args[i+1])
	}
	event.Msg(msg)
}

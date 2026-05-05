package nakama

import (
	"context"
	"errors"
	"hash/fnv"
	"math"
	"math/rand"
	"time"
)

// Retry represents a single retry attempt with the delay components used.
// It mirrors Nakama/Retry.cs.
type Retry struct {
	ExponentialBackoff time.Duration
	JitterBackoff      time.Duration
}

// Jitter is a function that introduces randomness into a retry delay.
// It mirrors Nakama/RetryJitter.cs.
type Jitter func(history []Retry, retryDelay time.Duration, rng *rand.Rand) time.Duration

// FullJitter selects a random delay between zero and retryDelay.
func FullJitter(history []Retry, retryDelay time.Duration, rng *rand.Rand) time.Duration {
	if retryDelay <= 0 || rng == nil {
		return retryDelay
	}
	return time.Duration(rng.Float64() * float64(retryDelay))
}

// RetryListener is invoked before a new retry attempt is made.
type RetryListener func(attempt int, retry Retry)

// RetryConfiguration describes the retry policy for a request.
// It mirrors Nakama/RetryConfiguration.cs.
type RetryConfiguration struct {
	BaseDelay   time.Duration
	MaxAttempts int
	Jitter      Jitter
	Listener    RetryListener
}

// NewRetryConfiguration creates a retry configuration with the supplied base
// delay (in milliseconds) and maximum number of attempts. Jitter defaults to
// FullJitter and listener to nil.
func NewRetryConfiguration(baseDelayMs, maxAttempts int) *RetryConfiguration {
	return &RetryConfiguration{
		BaseDelay:   time.Duration(baseDelayMs) * time.Millisecond,
		MaxAttempts: maxAttempts,
		Jitter:      FullJitter,
	}
}

// retryHistory tracks the state of a single retriable request.
type retryHistory struct {
	cfg     *RetryConfiguration
	retries []Retry
	rng     *rand.Rand
}

func newRetryHistory(seed string, cfg *RetryConfiguration) *retryHistory {
	h := fnv.New64a()
	_, _ = h.Write([]byte(seed))
	return &retryHistory{
		cfg: cfg,
		rng: rand.New(rand.NewSource(int64(h.Sum64()))),
	}
}

// retryInvoker is a port of Nakama/RetryInvoker.cs.
type retryInvoker struct {
	transient TransientErrorFunc
}

func newRetryInvoker(t TransientErrorFunc) *retryInvoker {
	if t == nil {
		t = defaultTransientErrorFunc
	}
	return &retryInvoker{transient: t}
}

// invoke calls request, retrying transient failures using history's
// configuration.
func (r *retryInvoker) invoke(ctx context.Context, history *retryHistory, request func(context.Context) error) error {
	for {
		err := request(ctx)
		if err == nil {
			return nil
		}
		if history == nil || history.cfg == nil || !r.transient(err) {
			return err
		}
		if backoffErr := r.backoff(ctx, history, err); backoffErr != nil {
			return backoffErr
		}
	}
}

// invokeT runs request, retrying transient failures, and returns a typed result.
func invokeT[T any](ctx context.Context, r *retryInvoker, history *retryHistory, request func(context.Context) (T, error)) (T, error) {
	var zero T
	for {
		v, err := request(ctx)
		if err == nil {
			return v, nil
		}
		if history == nil || history.cfg == nil || !r.transient(err) {
			return zero, err
		}
		if backoffErr := r.backoff(ctx, history, err); backoffErr != nil {
			return zero, backoffErr
		}
	}
}

func (r *retryInvoker) backoff(ctx context.Context, history *retryHistory, lastErr error) error {
	if len(history.retries) >= history.cfg.MaxAttempts {
		return errors.Join(errors.New("nakama: exceeded max retry attempts"), lastErr)
	}

	expo := time.Duration(math.Pow(2, float64(len(history.retries)))) * history.cfg.BaseDelay
	jit := expo
	if history.cfg.Jitter != nil {
		jit = history.cfg.Jitter(history.retries, expo, history.rng)
	}
	retry := Retry{ExponentialBackoff: expo, JitterBackoff: jit}
	history.retries = append(history.retries, retry)

	if history.cfg.Listener != nil {
		history.cfg.Listener(len(history.retries), retry)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	timer := time.NewTimer(jit)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}
	return nil
}

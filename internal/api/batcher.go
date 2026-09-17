package api

import (
	"context"
	"log"
	"time"

	"github.com/ashutoshk/url-shortener/internal/store"
)

// ClickBatcher decouples the redirect hot path from the write cost of
// logging analytics events. Redirects push into a buffered channel and
// return immediately; a background goroutine flushes accumulated events
// to Postgres in batches, either when the batch fills or on a timer —
// whichever comes first. This keeps p99 redirect latency independent of
// analytics write volume.
type ClickBatcher struct {
	store    *store.Store
	events   chan store.ClickEvent
	batch    []store.ClickEvent
	maxBatch int
	interval time.Duration
}

func NewClickBatcher(s *store.Store, bufferSize, maxBatch int, interval time.Duration) *ClickBatcher {
	return &ClickBatcher{
		store:    s,
		events:   make(chan store.ClickEvent, bufferSize),
		batch:    make([]store.ClickEvent, 0, maxBatch),
		maxBatch: maxBatch,
		interval: interval,
	}
}

// Enqueue never blocks the caller on a full buffer: it drops the event
// rather than back-pressuring the redirect response. Losing an
// occasional click event is an acceptable trade-off for never slowing
// down a redirect.
func (b *ClickBatcher) Enqueue(e store.ClickEvent) {
	select {
	case b.events <- e:
	default:
		log.Printf("click buffer full, dropping event for code=%s", e.Code)
	}
}

// Run drains the event channel until ctx is cancelled, then flushes any
// remaining buffered events before returning.
func (b *ClickBatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	for {
		select {
		case e := <-b.events:
			b.batch = append(b.batch, e)
			if len(b.batch) >= b.maxBatch {
				b.flush(ctx)
			}
		case <-ticker.C:
			b.flush(ctx)
		case <-ctx.Done():
			b.drain(ctx)
			return
		}
	}
}

func (b *ClickBatcher) drain(ctx context.Context) {
	for {
		select {
		case e := <-b.events:
			b.batch = append(b.batch, e)
		default:
			b.flush(context.Background())
			return
		}
	}
}

func (b *ClickBatcher) flush(ctx context.Context) {
	if len(b.batch) == 0 {
		return
	}
	if err := b.store.InsertClicks(ctx, b.batch); err != nil {
		log.Printf("failed to flush %d click events: %v", len(b.batch), err)
	}
	b.batch = b.batch[:0]
}

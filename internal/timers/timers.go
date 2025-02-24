package timers

import (
	"context"
	"time"

	"go.uber.org/zap"
)

const ContextKey = "timers"

type Timings struct {
	t     []time.Time
	names map[string]int
}

func New() *Timings {
	return &Timings{
		t:     make([]time.Time, 0, 10),
		names: make(map[string]int),
	}
}

func Checkpoint(ctx context.Context, name string) {
	t := ctx.Value(ContextKey)
	if t == nil {
		zap.L().Error("timers not initialized on context", zap.String("name", name))
		return
	}
	ti, ok := t.(*Timings)
	if !ok {
		zap.L().Error("timers not initialized correctly: not a timer")
		return
	}
	ti.t = append(ti.t, time.Now())
	ti.names[name] = len(ti.t) - 1
}

package context

import (
	"context"
	"time"
)

var _ context.Context = make(StopChannelContext)

// StopChannelContext implements context.Context wrapping an existing stop
// channel. This is useful for converting between methods that use a stop
// channel and libraries that accept a context.
type StopChannelContext <-chan struct{}

func (StopChannelContext) Deadline() (deadline time.Time, ok bool) {
	return
}

func (ctx StopChannelContext) Done() <-chan struct{} {
	return ctx
}

func (StopChannelContext) Err() error {
	return nil
}

func (StopChannelContext) Value(key any) any {
	return nil
}

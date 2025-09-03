package f

import (
	"context"

	"github.com/gookit/event"
)

func FireEvent(ctx context.Context, evt string, data map[string]any) {
	event.MustFire(evt, data)
}

func OnEvent(ctx context.Context, evt string, handler func(data map[string]any) error) {
	event.On(evt, event.ListenerFunc(func(e event.Event) error {
		return handler(e.Data())
	}), event.Normal)
}

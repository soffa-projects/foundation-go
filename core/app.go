package adapters

import (
	"context"
)

type App interface {
	Start(port int)
	Shutdown(ctx context.Context)
	Router() Router
}

type AppID struct {
	Name    string
	Version string
}

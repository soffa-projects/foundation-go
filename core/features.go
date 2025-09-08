package adapters

import (
	"io/fs"
)

type FeatureSpec struct {
	//ApplyPatch func(ctx context.Context, db DB, patch int) (bool, error)
}

type Feature struct {
	Name       string
	FS         fs.FS
	DependsOn  []Feature
	Init       func() error
	InitRoutes func(router Router)
	Operations []OperationFn
}

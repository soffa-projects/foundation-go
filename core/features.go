package f

import (
	"fmt"
	"io/fs"

	"github.com/soffa-projects/foundation-go/log"
)

type FeatureSpec struct {
	//ApplyPatch func(ctx context.Context, db DB, patch int) (bool, error)
}

type Feature struct {
	Name       string
	FS         fs.FS
	DependsOn  []Feature
	Init       func(env *ApplicationEnv) error
	InitRoutes func(router Router)
}

func checkFeatures(features ...Feature) []Feature {
	featureMap := make(map[string]bool, len(features))
	loadedFeatures := []Feature{}
	for _, f := range features {
		if f.Name == "" {
			log.Fatal("feature name is required")
		}
		if _, ok := featureMap[f.Name]; ok {
			log.Fatal("feature name %s is already registered", f.Name)
		}
		featureMap[f.Name] = true
		loadedFeatures = append(loadedFeatures, f)
	}
	// make sure dependencies are loaded
	for _, f := range loadedFeatures {
		for _, dep := range f.DependsOn {
			if _, ok := featureMap[dep.Name]; !ok {
				loadedFeatures = append(loadedFeatures, dep)
				featureMap[dep.Name] = true
			}
		}
	}
	return orderFeatures(loadedFeatures)
}

func orderFeatures(features []Feature) []Feature {
	// Map features by name for lookup
	featureMap := make(map[string]Feature, len(features))
	for _, f := range features {
		featureMap[f.Name] = f
	}

	visited := make(map[string]bool)
	temp := make(map[string]bool) // for cycle detection
	var ordered []Feature

	var visit func(string) error

	visit = func(name string) error {
		if visited[name] {
			return nil
		}
		if temp[name] {
			log.Fatal("cyclic dependency detected: %s", name)
		}

		temp[name] = true
		feat, ok := featureMap[name]
		if !ok {
			return fmt.Errorf("unknown dependency: %s", name)
		}

		for _, dep := range feat.DependsOn {
			if err := visit(dep.Name); err != nil {
				return err
			}
		}

		temp[name] = false
		visited[name] = true
		ordered = append(ordered, feat)
		return nil
	}

	// Visit all features
	for _, f := range features {
		if !visited[f.Name] {
			if err := visit(f.Name); err != nil {
				log.Fatal("error visiting feature: %s -- %v", f.Name, err)
				return nil
			}
		}
	}

	return ordered
}

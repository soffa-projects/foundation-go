package f

import (
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
	// Map features by name for quick lookup
	featureMap := make(map[string]Feature, len(features))
	indegree := make(map[string]int, len(features)) // number of dependencies
	graph := make(map[string][]string)              // adjacency list

	// Initialize maps
	for _, f := range features {
		featureMap[f.Name] = f
		if _, ok := indegree[f.Name]; !ok {
			indegree[f.Name] = 0
		}
	}

	// Build graph and indegree count
	for _, f := range features {
		for _, dep := range f.DependsOn {
			graph[dep.Name] = append(graph[dep.Name], f.Name)
			indegree[f.Name]++
		}
	}

	// Queue of features with no dependencies
	queue := make([]string, 0)
	for name, deg := range indegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}

	// Perform topological sort
	var ordered []Feature
	for len(queue) > 0 {
		// Pop from queue
		name := queue[0]
		queue = queue[1:]

		feat, ok := featureMap[name]
		if !ok {
			log.Fatal("unknown feature: %s", name)
		}
		ordered = append(ordered, feat)

		// Decrease indegree of dependents
		for _, neigh := range graph[name] {
			indegree[neigh]--
			if indegree[neigh] == 0 {
				queue = append(queue, neigh)
			}
		}
	}

	// Check for cycles
	if len(ordered) != len(features) {
		log.Fatal("cyclic dependency detected - %v  / %v", len(ordered), len(features))
	}

	return ordered
}

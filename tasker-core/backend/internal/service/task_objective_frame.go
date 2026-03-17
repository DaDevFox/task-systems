package service

import (
	"fmt"
	"sync"
)

// TaskDomain captures policy ownership and access boundaries for grouped tasks.
type TaskDomain struct {
	ID             string
	Name           string
	ReadGroup      string
	WriteGroup     string
	CompleteGroup  string
	ConfigureGroup string
}

// TaskFactoryFrame defines who can configure and dispatch tasks for a source.
type TaskFactoryFrame struct {
	ID                 string
	SourceTag          string
	ConfigureGroup     string
	AssignableUserList []string
}

// TaskResultRequirement describes a result that must be fulfilled before completion.
type TaskResultRequirement struct {
	ID          string
	Name        string
	Kind        string
	Required    bool
	Awaitable   bool
	Description string
}

// TaskResourceDependency models external resource dependencies (APIs, checks, etc).
type TaskResourceDependency struct {
	ID          string
	Kind        string
	Awaitable   bool
	Description string
}

// TraitDefinition describes a typed trait system can track/index.
type TraitDefinition struct {
	Name        string
	ValueType   string
	Description string
}

// ObjectiveFrame stores architecture-level definitions for review and future implementation.
type ObjectiveFrame struct {
	mu          sync.RWMutex
	domains     map[string]TaskDomain
	factories   map[string]TaskFactoryFrame
	results     map[string]TaskResultRequirement
	resources   map[string]TaskResourceDependency
	traits      map[string]TraitDefinition
	systemNames map[string]struct{}
}

func NewObjectiveFrame() *ObjectiveFrame {
	return &ObjectiveFrame{
		domains:     map[string]TaskDomain{},
		factories:   map[string]TaskFactoryFrame{},
		results:     map[string]TaskResultRequirement{},
		resources:   map[string]TaskResourceDependency{},
		traits:      map[string]TraitDefinition{},
		systemNames: map[string]struct{}{},
	}
}

func (f *ObjectiveFrame) DefineDomain(domain TaskDomain) error {
	if domain.ID == "" {
		return fmt.Errorf("domain id cannot be empty")
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.domains[domain.ID] = domain
	return nil
}

func (f *ObjectiveFrame) DefineFactory(factory TaskFactoryFrame) error {
	if factory.ID == "" {
		return fmt.Errorf("factory id cannot be empty")
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.factories[factory.ID] = factory
	return nil
}

func (f *ObjectiveFrame) DefineResult(result TaskResultRequirement) error {
	if result.ID == "" {
		return fmt.Errorf("result id cannot be empty")
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.results[result.ID] = result
	return nil
}

func (f *ObjectiveFrame) DefineResource(resource TaskResourceDependency) error {
	if resource.ID == "" {
		return fmt.Errorf("resource id cannot be empty")
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.resources[resource.ID] = resource
	return nil
}

func (f *ObjectiveFrame) DefineTrait(trait TraitDefinition) error {
	if trait.Name == "" {
		return fmt.Errorf("trait name cannot be empty")
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.traits[trait.Name] = trait
	return nil
}

func (f *ObjectiveFrame) RegisterSystemName(name string) error {
	if name == "" {
		return fmt.Errorf("system name cannot be empty")
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.systemNames[name] = struct{}{}
	return nil
}

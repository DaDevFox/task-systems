package domain

// ResultValue mirrors result.proto payload variants.
type ResultValue struct {
	FormOpenURL      string
	FormResponseURL  string
	FileLocationPath string
}

// TaskResult tracks completion against a requirement in a task.
type TaskResult struct {
	RequirementID string
	Result        ResultValue
	Complete      bool
}

// ResourceDependency mirrors resource.proto variants.
type ResourceDependency struct {
	APIURL             string
	APIResponseRegex   string
	FileLocationPath   string
	DependencyTypeHint string
}

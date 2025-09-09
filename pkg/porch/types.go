package porch

<<<<<<< HEAD
import "encoding/json"

// PorchError represents an error from the Porch API.

type PorchError struct {
	Code int `json:"code"`

	Message string `json:"message"`

	Details string `json:"details,omitempty"`
}

// Error performs error operation.

=======
// PorchError represents an error from the Porch API
type PorchError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

>>>>>>> 6835433495e87288b95961af7173d866977175ff
func (e *PorchError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
<<<<<<< HEAD

	return e.Message
}

// PackageSpec represents the specification of a package.

type PackageSpec struct {
	Repository string `json:"repository"`

	Package string `json:"package"`

	Workspace string `json:"workspace"`

	Resources json.RawMessage `json:"resources,omitempty"`
}

// PackageStatus represents the status of a package.

type PackageStatus struct {
	Phase string `json:"phase"`

	Message string `json:"message,omitempty"`

	Conditions []Condition `json:"conditions,omitempty"`
}

// Condition represents a condition in the package status.

type Condition struct {
	Type string `json:"type"`

	Status string `json:"status"`

	LastTransitionTime string `json:"lastTransitionTime,omitempty"`

	Reason string `json:"reason,omitempty"`

	Message string `json:"message,omitempty"`
}
=======
	return e.Message
}

// PackageSpec represents the specification of a package
type PackageSpec struct {
	Repository string                 `json:"repository"`
	Package    string                 `json:"package"`
	Workspace  string                 `json:"workspace"`
	Resources  map[string]interface{} `json:"resources,omitempty"`
}

// PackageStatus represents the status of a package
type PackageStatus struct {
	Phase      string `json:"phase"`
	Message    string `json:"message,omitempty"`
	Conditions []Condition `json:"conditions,omitempty"`
}

// Condition represents a condition in the package status
type Condition struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	Reason             string `json:"reason,omitempty"`
	Message            string `json:"message,omitempty"`
}

// WorkloadCluster represents a workload cluster for package deployment
type WorkloadCluster struct {
	Name         string            `json:"name"`
	Endpoint     string            `json:"endpoint"`
	Region       string            `json:"region"`
	Zone         string            `json:"zone"`
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
}
>>>>>>> 6835433495e87288b95961af7173d866977175ff

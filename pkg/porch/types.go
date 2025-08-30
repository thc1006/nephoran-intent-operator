package porch

// PorchError represents an error from the Porch API.

type PorchError struct {
	Code int `json:"code"`

	Message string `json:"message"`

	Details string `json:"details,omitempty"`
}

// Error performs error operation.

func (e *PorchError) Error() string {

	if e.Details != "" {

		return e.Message + ": " + e.Details

	}

	return e.Message

}

// PackageSpec represents the specification of a package.

type PackageSpec struct {
	Repository string `json:"repository"`

	Package string `json:"package"`

	Workspace string `json:"workspace"`

	Resources map[string]interface{} `json:"resources,omitempty"`
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

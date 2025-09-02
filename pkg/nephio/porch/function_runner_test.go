package porch

// Imports are left minimal since this is just a placeholder test file

// createTestFunctionRequest returns a test function request
func createTestFunctionRequest() *FunctionRequest {
	return &FunctionRequest{
		FunctionConfig: FunctionConfig{
			Image: "gcr.io/kpt-fn/apply-setters:v0.1.1",
			ConfigMap: map[string]interface{}{
				"key1": "value1",
			},
		},
		Resources: []KRMResource{
			{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Metadata: map[string]interface{}{
					"name":      "test-config",
					"namespace": "default",
				},
				Data: map[string]interface{}{
					"config": "test-value",
				},
			},
		},
		Context: &FunctionContext{
			Environment: map[string]string{
				"ENV_VAR": "test-value",
			},
		},
	}
}

// Rest of the function_runner_test.go follows the previous implementation...
// The only change is the createTestFunctionRequest implementation to use KRMResource
// instead of map[string]interface{}

import re

def fix_file():
    with open('pkg/nephio/krm/porch_integration.go', 'r') as f:
        content = f.read()
    
    # 1. Comment out all intent.Spec.Extensions references since they don't exist
    content = re.sub(r'(\s+)if intent\.Spec\.Extensions', r'\1// if intent.Spec.Extensions', content)
    
    # 2. Fix TargetComponent to use first element of TargetComponents properly
    content = content.replace('intent.Spec.TargetComponent', 'func() string { if len(intent.Spec.TargetComponents) > 0 { return string(intent.Spec.TargetComponents[0]) }; return "" }()')
    
    # 3. Fix FinalResources to OutputResources
    content = content.replace('pipelineExecution.FinalResources', 'pipelineExecution.OutputResources')
    content = content.replace('execution.FinalResources', 'execution.OutputResources')
    
    # 4. Fix ExecutedStages - replace with extracting stage names
    content = content.replace('pipelineExecution.ExecutedStages', 'func() []string { stages := make([]string, 0, len(pipelineExecution.Stages)); for name := range pipelineExecution.Stages { stages = append(stages, name) }; return stages }()')
    content = content.replace('execution.ExecutedStages', 'func() []string { stages := make([]string, 0, len(execution.Stages)); for name := range execution.Stages { stages = append(stages, name) }; return stages }()')
    
    # 5. Fix Resources type mismatch
    content = content.replace('Resources:   []porch.KRMResource{}', 'Resources: make([]interface{}, 0)')
    content = content.replace('updatedResources := make([]porch.KRMResource, 0)', 'updatedResources := make([]interface{}, 0)')
    
    # 6. Fix the is5GCoreIntent function
    old_function = '''func (pim *PorchIntegrationManager) is5GCoreIntent(intent *v1.NetworkIntent) bool {
	// Check if intent targets 5G Core components
	coreComponents := []string{"amf", "smf", "upf", "nssf", "nrf", "udm", "ausf", "pcf"}
	for _, component := range coreComponents {
		if func() string { if len(intent.Spec.TargetComponents) > 0 { return string(intent.Spec.TargetComponents[0]) }; return "" }() == component {
			return true
		}
	}
	return false
}'''

    new_function = '''func (pim *PorchIntegrationManager) is5GCoreIntent(intent *v1.NetworkIntent) bool {
	// Check if intent targets 5G Core components
	coreComponents := []string{"amf", "smf", "upf", "nssf", "nrf", "udm", "ausf", "pcf"}
	for _, component := range coreComponents {
		for _, targetComp := range intent.Spec.TargetComponents {
			if string(targetComp) == component {
				return true
			}
		}
	}
	return false
}'''
    
    content = content.replace(old_function, new_function)
    
    with open('pkg/nephio/krm/porch_integration.go', 'w') as f:
        f.write(content)

if __name__ == '__main__':
    fix_file()
    print("Fixed porch_integration.go")

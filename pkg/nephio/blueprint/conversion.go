package blueprint

import v1 "github.com/thc1006/nephoran-intent-operator/api/v1"

// convertNetworkTargetComponentsToORANComponents converts NetworkTargetComponent to ORANComponent
func convertNetworkTargetComponentsToORANComponents(components []v1.NetworkTargetComponent) []v1.ORANComponent {
	var oranComponents []v1.ORANComponent
	for _, component := range components {
		switch component {
		case v1.NetworkTargetComponentAMF:
			oranComponents = append(oranComponents, v1.ORANComponentAMF)
		case v1.NetworkTargetComponentSMF:
			oranComponents = append(oranComponents, v1.ORANComponent("smf"))
		case v1.NetworkTargetComponentUPF:
			oranComponents = append(oranComponents, v1.ORANComponent("upf"))
		case v1.NetworkTargetComponentNearRTRIC:
			oranComponents = append(oranComponents, v1.ORANComponentNearRTRIC)
		case v1.NetworkTargetComponentXApp:
			oranComponents = append(oranComponents, v1.ORANComponentXApp)
		default:
			// For unknown components, try to cast directly
			oranComponents = append(oranComponents, v1.ORANComponent(component))
		}
	}
	return oranComponents
}

// convertNetworkTargetComponentToORANComponent converts a single NetworkTargetComponent to ORANComponent
func convertNetworkTargetComponentToORANComponent(component v1.NetworkTargetComponent) v1.ORANComponent {
	switch component {
	case v1.NetworkTargetComponentAMF:
		return v1.ORANComponentAMF
	case v1.NetworkTargetComponentSMF:
		return v1.ORANComponent("smf")
	case v1.NetworkTargetComponentUPF:
		return v1.ORANComponent("upf")
	case v1.NetworkTargetComponentNearRTRIC:
		return v1.ORANComponentNearRTRIC
	case v1.NetworkTargetComponentXApp:
		return v1.ORANComponentXApp
	default:
		return v1.ORANComponent(component)
	}
}

// networkTargetComponentsToStrings converts NetworkTargetComponents to string slice
func networkTargetComponentsToStrings(components []v1.NetworkTargetComponent) []string {
	strings := make([]string, len(components))
	for i, component := range components {
		strings[i] = string(component)
	}
	return strings
}

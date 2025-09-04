/*

Copyright 2025.



Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.

You may obtain a copy of the License at



    http://www.apache.org/licenses/LICENSE-2.0



Unless required by applicable law or agreed to in writing, software

distributed under the License is distributed on an "AS IS" BASIS,

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions and

limitations under the License.

*/

package shared

import (
	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
)

// ProcessingPhaseToNetworkIntentPhase converts a ProcessingPhase to NetworkIntentPhase.

func ProcessingPhaseToNetworkIntentPhase(phase interfaces.ProcessingPhase) nephoranv1.NetworkIntentPhase {
	switch phase {

	case interfaces.PhaseIntentReceived, interfaces.PhaseReceived:

		return nephoranv1.NetworkIntentPhasePending

	case interfaces.PhaseLLMProcessing:

		return nephoranv1.NetworkIntentPhaseProcessing

	case interfaces.PhaseResourcePlanning, interfaces.PhaseManifestGeneration:

		return nephoranv1.NetworkIntentPhaseDeploying

	case interfaces.PhaseGitOpsCommit:

		return nephoranv1.NetworkIntentPhaseDeploying

	case interfaces.PhaseDeploymentVerification:

		return nephoranv1.NetworkIntentPhaseActive

	case interfaces.PhaseCompleted:

		return nephoranv1.NetworkIntentPhaseDeployed

	case interfaces.PhaseFailed:

		return nephoranv1.NetworkIntentPhaseFailed

	default:

		return nephoranv1.NetworkIntentPhasePending

	}
}

// NetworkIntentPhaseToProcessingPhase converts a NetworkIntentPhase to ProcessingPhase.

func NetworkIntentPhaseToProcessingPhase(phase nephoranv1.NetworkIntentPhase) interfaces.ProcessingPhase {
	switch phase {

	case nephoranv1.NetworkIntentPhasePending:

		return interfaces.PhaseIntentReceived

	case nephoranv1.NetworkIntentPhaseProcessing:

		return interfaces.PhaseLLMProcessing

	case nephoranv1.NetworkIntentPhaseDeploying:

		return interfaces.PhaseResourcePlanning

	case nephoranv1.NetworkIntentPhaseActive:

		return interfaces.PhaseDeploymentVerification

	case nephoranv1.NetworkIntentPhaseReady, nephoranv1.NetworkIntentPhaseDeployed:

		return interfaces.PhaseCompleted

	case nephoranv1.NetworkIntentPhaseFailed:

		return interfaces.PhaseFailed

	default:

		return interfaces.PhaseIntentReceived

	}
}

// StringToNetworkIntentPhase converts a string to NetworkIntentPhase with validation.

func StringToNetworkIntentPhase(phase string) nephoranv1.NetworkIntentPhase {
	switch phase {

	case "Pending":

		return nephoranv1.NetworkIntentPhasePending

	case "Processing":

		return nephoranv1.NetworkIntentPhaseProcessing

	case "Deploying":

		return nephoranv1.NetworkIntentPhaseDeploying

	case "Active":

		return nephoranv1.NetworkIntentPhaseActive

	case "Ready":

		return nephoranv1.NetworkIntentPhaseReady

	case "Deployed":

		return nephoranv1.NetworkIntentPhaseDeployed

	case "Failed":

		return nephoranv1.NetworkIntentPhaseFailed

	default:

		return nephoranv1.NetworkIntentPhasePending

	}
}

// NetworkIntentPhaseToString converts a NetworkIntentPhase to string.

func NetworkIntentPhaseToString(phase nephoranv1.NetworkIntentPhase) string {
	return string(phase)
}

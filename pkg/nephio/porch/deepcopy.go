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

package porch

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeepCopy implementations for all Porch types
// These methods provide type-safe deep copying required for Kubernetes client-go

// Repository DeepCopy methods

func (r *RepositoryList) DeepCopy() *RepositoryList {
	if r == nil {
		return nil
	}
	out := new(RepositoryList)
	r.DeepCopyInto(out)
	return out
}

func (r *RepositoryList) DeepCopyInto(out *RepositoryList) {
	*out = *r
	out.TypeMeta = r.TypeMeta
	r.ListMeta.DeepCopyInto(&out.ListMeta)
	if r.Items != nil {
		out.Items = make([]Repository, len(r.Items))
		for i := range r.Items {
			r.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

func (ac *AuthConfig) DeepCopy() *AuthConfig {
	if ac == nil {
		return nil
	}
	out := new(AuthConfig)
	ac.DeepCopyInto(out)
	return out
}

func (ac *AuthConfig) DeepCopyInto(out *AuthConfig) {
	*out = *ac
	if ac.SecretRef != nil {
		out.SecretRef = new(SecretReference)
		*out.SecretRef = *ac.SecretRef
	}
	if ac.Headers != nil {
		out.Headers = make(map[string]string, len(ac.Headers))
		for k, v := range ac.Headers {
			out.Headers[k] = v
		}
	}
}

func (sc *SyncConfig) DeepCopy() *SyncConfig {
	if sc == nil {
		return nil
	}
	out := new(SyncConfig)
	sc.DeepCopyInto(out)
	return out
}

func (sc *SyncConfig) DeepCopyInto(out *SyncConfig) {
	*out = *sc
	if sc.Interval != nil {
		out.Interval = new(metav1.Duration)
		*out.Interval = *sc.Interval
	}
	if sc.BackoffLimit != nil {
		out.BackoffLimit = new(metav1.Duration)
		*out.BackoffLimit = *sc.BackoffLimit
	}
}

// PackageRevision DeepCopy methods

// DeepCopy methods for PackageRevision are in multicluster package

// PackageRevisionSpec DeepCopy methods are in multicluster package

// PackageRevisionStatus DeepCopy methods are in multicluster package

// Note: PackageRevision-related DeepCopy methods moved to multicluster package

// KRMResource DeepCopy methods

func (kr *KRMResource) DeepCopyInto(out *KRMResource) {
	*out = *kr
	if kr.Metadata != nil {
		out.Metadata = make(map[string]interface{}, len(kr.Metadata))
		for k, v := range kr.Metadata {
			out.Metadata[k] = deepCopyInterface(v)
		}
	}
	if kr.Spec != nil {
		out.Spec = make(map[string]interface{}, len(kr.Spec))
		for k, v := range kr.Spec {
			out.Spec[k] = deepCopyInterface(v)
		}
	}
	if kr.Status != nil {
		out.Status = make(map[string]interface{}, len(kr.Status))
		for k, v := range kr.Status {
			out.Status[k] = deepCopyInterface(v)
		}
	}
	if kr.Data != nil {
		out.Data = make(map[string]interface{}, len(kr.Data))
		for k, v := range kr.Data {
			out.Data[k] = deepCopyInterface(v)
		}
	}
}

// FunctionConfig DeepCopy methods

func (fc *FunctionConfig) DeepCopyInto(out *FunctionConfig) {
	*out = *fc
	if fc.ConfigMap != nil {
		out.ConfigMap = make(map[string]interface{}, len(fc.ConfigMap))
		for k, v := range fc.ConfigMap {
			out.ConfigMap[k] = deepCopyInterface(v)
		}
	}
	if fc.Selectors != nil {
		out.Selectors = make([]ResourceSelector, len(fc.Selectors))
		for i := range fc.Selectors {
			fc.Selectors[i].DeepCopyInto(&out.Selectors[i])
		}
	}
	if fc.Exec != nil {
		out.Exec = new(ExecConfig)
		fc.Exec.DeepCopyInto(out.Exec)
	}
}

func (rs *ResourceSelector) DeepCopyInto(out *ResourceSelector) {
	*out = *rs
	if rs.Labels != nil {
		out.Labels = make(map[string]string, len(rs.Labels))
		for k, v := range rs.Labels {
			out.Labels[k] = v
		}
	}
}

func (ec *ExecConfig) DeepCopyInto(out *ExecConfig) {
	*out = *ec
	if ec.Args != nil {
		out.Args = make([]string, len(ec.Args))
		copy(out.Args, ec.Args)
	}
	if ec.Env != nil {
		out.Env = make(map[string]string, len(ec.Env))
		for k, v := range ec.Env {
			out.Env[k] = v
		}
	}
}

// Workflow DeepCopy methods

func (w *Workflow) DeepCopy() *Workflow {
	if w == nil {
		return nil
	}
	out := new(Workflow)
	w.DeepCopyInto(out)
	return out
}

func (w *Workflow) DeepCopyInto(out *Workflow) {
	*out = *w
	out.TypeMeta = w.TypeMeta
	w.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	w.Spec.DeepCopyInto(&out.Spec)
	w.Status.DeepCopyInto(&out.Status)
}

func (ws *WorkflowSpec) DeepCopyInto(out *WorkflowSpec) {
	*out = *ws
	if ws.Stages != nil {
		out.Stages = make([]WorkflowStage, len(ws.Stages))
		for i := range ws.Stages {
			ws.Stages[i].DeepCopyInto(&out.Stages[i])
		}
	}
	if ws.Triggers != nil {
		out.Triggers = make([]WorkflowTrigger, len(ws.Triggers))
		for i := range ws.Triggers {
			ws.Triggers[i].DeepCopyInto(&out.Triggers[i])
		}
	}
	if ws.Approvers != nil {
		out.Approvers = make([]Approver, len(ws.Approvers))
		copy(out.Approvers, ws.Approvers)
	}
	if ws.Timeout != nil {
		out.Timeout = new(metav1.Duration)
		*out.Timeout = *ws.Timeout
	}
	if ws.RetryPolicy != nil {
		out.RetryPolicy = new(RetryPolicy)
		ws.RetryPolicy.DeepCopyInto(out.RetryPolicy)
	}
}

func (ws *WorkflowStatus) DeepCopyInto(out *WorkflowStatus) {
	*out = *ws
	if ws.Conditions != nil {
		out.Conditions = make([]metav1.Condition, len(ws.Conditions))
		for i := range ws.Conditions {
			ws.Conditions[i].DeepCopyInto(&out.Conditions[i])
		}
	}
	if ws.StartTime != nil {
		out.StartTime = new(metav1.Time)
		ws.StartTime.DeepCopyInto(out.StartTime)
	}
	if ws.EndTime != nil {
		out.EndTime = new(metav1.Time)
		ws.EndTime.DeepCopyInto(out.EndTime)
	}
	if ws.Results != nil {
		out.Results = make([]WorkflowResult, len(ws.Results))
		for i := range ws.Results {
			ws.Results[i].DeepCopyInto(&out.Results[i])
		}
	}
}

func (wl *WorkflowList) DeepCopy() *WorkflowList {
	if wl == nil {
		return nil
	}
	out := new(WorkflowList)
	wl.DeepCopyInto(out)
	return out
}

func (wl *WorkflowList) DeepCopyInto(out *WorkflowList) {
	*out = *wl
	out.TypeMeta = wl.TypeMeta
	wl.ListMeta.DeepCopyInto(&out.ListMeta)
	if wl.Items != nil {
		out.Items = make([]Workflow, len(wl.Items))
		for i := range wl.Items {
			wl.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

// Supporting types DeepCopy methods

func (pm *PackageMetadata) DeepCopyInto(out *PackageMetadata) {
	*out = *pm
	if pm.Keywords != nil {
		out.Keywords = make([]string, len(pm.Keywords))
		copy(out.Keywords, pm.Keywords)
	}
	if pm.Labels != nil {
		out.Labels = make(map[string]string, len(pm.Labels))
		for k, v := range pm.Labels {
			out.Labels[k] = v
		}
	}
	if pm.Annotations != nil {
		out.Annotations = make(map[string]string, len(pm.Annotations))
		for k, v := range pm.Annotations {
			out.Annotations[k] = v
		}
	}
}

func (wl *WorkflowLock) DeepCopyInto(out *WorkflowLock) {
	*out = *wl
	if wl.LockedAt != nil {
		out.LockedAt = new(metav1.Time)
		wl.LockedAt.DeepCopyInto(out.LockedAt)
	}
}

func (ds *DeploymentStatus) DeepCopyInto(out *DeploymentStatus) {
	*out = *ds
	if ds.Conditions != nil {
		out.Conditions = make([]metav1.Condition, len(ds.Conditions))
		for i := range ds.Conditions {
			ds.Conditions[i].DeepCopyInto(&out.Conditions[i])
		}
	}
	if ds.Targets != nil {
		out.Targets = make([]DeploymentTarget, len(ds.Targets))
		for i := range ds.Targets {
			ds.Targets[i].DeepCopyInto(&out.Targets[i])
		}
	}
	if ds.StartTime != nil {
		out.StartTime = new(metav1.Time)
		ds.StartTime.DeepCopyInto(out.StartTime)
	}
	if ds.EndTime != nil {
		out.EndTime = new(metav1.Time)
		ds.EndTime.DeepCopyInto(out.EndTime)
	}
}

func (dt *DeploymentTarget) DeepCopyInto(out *DeploymentTarget) {
	*out = *dt
	if dt.Resources != nil {
		out.Resources = make([]DeployedResource, len(dt.Resources))
		copy(out.Resources, dt.Resources)
	}
}

func (vr *ValidationResult) DeepCopyInto(out *ValidationResult) {
	*out = *vr
	if vr.Errors != nil {
		out.Errors = make([]ValidationError, len(vr.Errors))
		copy(out.Errors, vr.Errors)
	}
	if vr.Warnings != nil {
		out.Warnings = make([]ValidationError, len(vr.Warnings))
		copy(out.Warnings, vr.Warnings)
	}
}

func (rr *RenderResult) DeepCopyInto(out *RenderResult) {
	*out = *rr
	if rr.Resources != nil {
		out.Resources = make([]KRMResource, len(rr.Resources))
		for i := range rr.Resources {
			rr.Resources[i].DeepCopyInto(&out.Resources[i])
		}
	}
	if rr.Results != nil {
		out.Results = make([]*FunctionResult, len(rr.Results))
		for i := range rr.Results {
			if rr.Results[i] != nil {
				out.Results[i] = new(FunctionResult)
				*out.Results[i] = *rr.Results[i]
				if rr.Results[i].Tags != nil {
					out.Results[i].Tags = make(map[string]string, len(rr.Results[i].Tags))
					for k, v := range rr.Results[i].Tags {
						out.Results[i].Tags[k] = v
					}
				}
			}
		}
	}
	if rr.Error != nil {
		out.Error = new(RenderError)
		*out.Error = *rr.Error
	}
}

func (ws *WorkflowStage) DeepCopyInto(out *WorkflowStage) {
	*out = *ws
	if ws.Conditions != nil {
		out.Conditions = make([]WorkflowCondition, len(ws.Conditions))
		for i := range ws.Conditions {
			ws.Conditions[i].DeepCopyInto(&out.Conditions[i])
		}
	}
	if ws.Actions != nil {
		out.Actions = make([]WorkflowAction, len(ws.Actions))
		for i := range ws.Actions {
			ws.Actions[i].DeepCopyInto(&out.Actions[i])
		}
	}
	if ws.Approvers != nil {
		out.Approvers = make([]Approver, len(ws.Approvers))
		copy(out.Approvers, ws.Approvers)
	}
	if ws.Timeout != nil {
		out.Timeout = new(metav1.Duration)
		*out.Timeout = *ws.Timeout
	}
	if ws.OnFailure != nil {
		out.OnFailure = new(FailureAction)
		ws.OnFailure.DeepCopyInto(out.OnFailure)
	}
}

func (wt *WorkflowTrigger) DeepCopyInto(out *WorkflowTrigger) {
	*out = *wt
	if wt.Condition != nil {
		out.Condition = make(map[string]interface{}, len(wt.Condition))
		for k, v := range wt.Condition {
			out.Condition[k] = deepCopyInterface(v)
		}
	}
}

func (rp *RetryPolicy) DeepCopyInto(out *RetryPolicy) {
	*out = *rp
	if rp.BackoffDelay != nil {
		out.BackoffDelay = new(metav1.Duration)
		*out.BackoffDelay = *rp.BackoffDelay
	}
}

func (wc *WorkflowCondition) DeepCopyInto(out *WorkflowCondition) {
	*out = *wc
	if wc.Condition != nil {
		out.Condition = make(map[string]interface{}, len(wc.Condition))
		for k, v := range wc.Condition {
			out.Condition[k] = deepCopyInterface(v)
		}
	}
}

func (wa *WorkflowAction) DeepCopyInto(out *WorkflowAction) {
	*out = *wa
	if wa.Config != nil {
		out.Config = make(map[string]interface{}, len(wa.Config))
		for k, v := range wa.Config {
			out.Config[k] = deepCopyInterface(v)
		}
	}
}

func (fa *FailureAction) DeepCopyInto(out *FailureAction) {
	*out = *fa
	if fa.Config != nil {
		out.Config = make(map[string]interface{}, len(fa.Config))
		for k, v := range fa.Config {
			out.Config[k] = deepCopyInterface(v)
		}
	}
}

func (wr *WorkflowResult) DeepCopyInto(out *WorkflowResult) {
	*out = *wr
	if wr.Timestamp != nil {
		out.Timestamp = new(metav1.Time)
		wr.Timestamp.DeepCopyInto(out.Timestamp)
	}
	if wr.Data != nil {
		out.Data = make(map[string]interface{}, len(wr.Data))
		for k, v := range wr.Data {
			out.Data[k] = deepCopyInterface(v)
		}
	}
}

// Network slice types DeepCopy methods

func (sr *SliceResources) DeepCopyInto(out *SliceResources) {
	*out = *sr
	if sr.CPU != nil {
		out.CPU = new(resource.Quantity)
		*out.CPU = sr.CPU.DeepCopy()
	}
	if sr.Memory != nil {
		out.Memory = new(resource.Quantity)
		*out.Memory = sr.Memory.DeepCopy()
	}
	if sr.Storage != nil {
		out.Storage = new(resource.Quantity)
		*out.Storage = sr.Storage.DeepCopy()
	}
	if sr.Bandwidth != nil {
		out.Bandwidth = new(resource.Quantity)
		*out.Bandwidth = sr.Bandwidth.DeepCopy()
	}
}

func (lr *LatencyRequirement) DeepCopyInto(out *LatencyRequirement) {
	*out = *lr
	if lr.MaxLatency != nil {
		out.MaxLatency = new(metav1.Duration)
		*out.MaxLatency = *lr.MaxLatency
	}
	if lr.TypicalLatency != nil {
		out.TypicalLatency = new(metav1.Duration)
		*out.TypicalLatency = *lr.TypicalLatency
	}
}

func (tr *ThroughputRequirement) DeepCopyInto(out *ThroughputRequirement) {
	*out = *tr
	if tr.MinDownlink != nil {
		out.MinDownlink = new(resource.Quantity)
		*out.MinDownlink = tr.MinDownlink.DeepCopy()
	}
	if tr.MinUplink != nil {
		out.MinUplink = new(resource.Quantity)
		*out.MinUplink = tr.MinUplink.DeepCopy()
	}
	if tr.MaxDownlink != nil {
		out.MaxDownlink = new(resource.Quantity)
		*out.MaxDownlink = tr.MaxDownlink.DeepCopy()
	}
	if tr.MaxUplink != nil {
		out.MaxUplink = new(resource.Quantity)
		*out.MaxUplink = tr.MaxUplink.DeepCopy()
	}
}

func (ar *AvailabilityRequirement) DeepCopyInto(out *AvailabilityRequirement) {
	*out = *ar
	if ar.Downtime != nil {
		out.Downtime = new(metav1.Duration)
		*out.Downtime = *ar.Downtime
	}
}

func (rr *ReliabilityRequirement) DeepCopyInto(out *ReliabilityRequirement) {
	*out = *rr
	if rr.MeanTimeBetweenFailures != nil {
		out.MeanTimeBetweenFailures = new(metav1.Duration)
		*out.MeanTimeBetweenFailures = *rr.MeanTimeBetweenFailures
	}
	if rr.MeanTimeToRepair != nil {
		out.MeanTimeToRepair = new(metav1.Duration)
		*out.MeanTimeToRepair = *rr.MeanTimeToRepair
	}
}

// deepCopyInterface provides deep copying for interface{} values
func deepCopyInterface(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case map[string]interface{}:
		copy := make(map[string]interface{}, len(val))
		for k, v := range val {
			copy[k] = deepCopyInterface(v)
		}
		return copy
	case []interface{}:
		copy := make([]interface{}, len(val))
		for i, v := range val {
			copy[i] = deepCopyInterface(v)
		}
		return copy
	case string, int, int32, int64, float32, float64, bool:
		return val
	default:
		// For unknown types, return as-is (this may not be safe for all types)
		return val
	}
}

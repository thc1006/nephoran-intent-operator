package v1alpha1

import (
"k8s.io/apimachinery/pkg/runtime/schema"
"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
// GroupVersion defines the API group and version
GroupVersion = schema.GroupVersion{Group: "intent.nephoran.com", Version: "v1alpha1"}

// SchemeBuilder 用來把型別註冊到 Scheme
SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

// AddToScheme 供 main.go 呼叫註冊
AddToScheme = SchemeBuilder.AddToScheme
)

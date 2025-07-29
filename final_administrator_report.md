# Kubernetes Cluster Issue Report: "resource mapping not found" for new CRD

## Problem Summary

We are attempting to deploy a new Kubernetes controller (`nephio-bridge`) that relies on a Custom Resource Definition (CRD), `E2NodeSet`. While the CRD appears to be successfully applied to the cluster, the Kubernetes API server does not recognize the new `E2NodeSet` resource type.

This failure prevents us from creating instances of the `E2NodeSet` resource, which blocks the deployment and testing of our controller. The error message is consistently: `error: resource mapping not found for name: "..." no matches for kind "E2NodeSet"`.

## Actions Taken

1.  **CRD Defined:** A new CRD, `E2NodeSet` (`e2nodesets.nephoran.com`), was defined using standard `controller-tools`.
2.  **CRD Applied:** The CRD manifest was successfully applied to the cluster using `kubectl apply`. The command completed without error and reported that the CRD was created or configured.
3.  **Verification Failed:** Subsequent attempts to create a custom resource of kind `E2NodeSet` fail with the "resource mapping not found" error.
4.  **Troubleshooting:** We have performed extensive troubleshooting, including:
    *   Verifying the CRD and custom resource YAML for correctness.
    *   Confirming the `nephio-bridge` controller can connect to the API server.
    *   Adding significant delays between applying the CRD and the custom resource to account for potential caching issues.
    *   The issue persists despite these efforts.

## Diagnostic Evidence

The following information was gathered using the attached `diagnose_cluster.sh` script.

### 1. Output of `kubectl get crd e2nodesets.nephoran.com -o yaml`
*(This confirms the CRD is present in the cluster's storage.)*
```
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.14.0
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apiextensions.k8s.io/v1","kind":"CustomResourceDefinition","metadata":{"annotations":{"controller-gen.kubebuilder.io/version":"v0.14.0"},"name":"e2nodesets.nephoran.com"},"spec":{"group":"nephoran.com","names":{"kind":"E2NodeSet","listKind":"E2NodeSetList","plural":"e2nodesets","singular":"e2nodeset"},"scope":"Namespaced","versions":[{"name":"v1alpha1","schema":{"openAPIV3Schema":{"description":"E2NodeSet is the Schema for the e2nodesets API","properties":{"apiVersion":{"description":"APIVersion defines the versioned schema of this representation of an object.\nServers should convert recognized schemas to the latest internal value, and\nmay reject unrecognized values.\nMore info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources","type":"string"},"kind":{"description":"Kind is a string value representing the REST resource this object represents.\nServers may infer this from the endpoint the client submits requests to.\nCannot be updated.\nIn CamelCase.\nMore info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds","type":"string"},"metadata":{"type":"object"},"spec":{"description":"E2NodeSetSpec defines the desired state of E2NodeSet","properties":{"replicas":{"description":"Replicas is the number of simulated E2 Nodes to run.","format":"int32","minimum":0,"type":"integer"}},"required":["replicas"],"type":"object"},"status":{"description":"E2NodeSetStatus defines the observed state of E2NodeSet","properties":{"readyReplicas":{"description":"ReadyReplicas is the number of E2 Nodes that are ready.","format":"int32","type":"integer"}},"type":"object"}},"type":"object"}},"served":true,"storage":true,"subresources":{"status":{}}}]}}
  creationTimestamp: "2025-07-26T17:48:45Z"
  generation: 1
  name: e2nodesets.nephoran.com
  resourceVersion: "2048427"
  uid: 34875d95-5fd4-46d0-bd75-f2cdfedd6852
spec:
  conversion:
    strategy: None
  group: nephoran.com
  names:
    kind: E2NodeSet
    listKind: E2NodeSetList
    plural: e2nodesets
    singular: e2nodeset
  scope: Namespaced
  versions:
  - name: v1alpha1
    schema:
      openAPIV3Schema:
        description: E2NodeSet is the Schema for the e2nodesets API
        properties:
          apiVersion:
            description: |-
              APIVersion defines the versioned schema of this representation of an object.
              Servers should convert recognized schemas to the latest internal value, and
              may reject unrecognized values.
              More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources
            type: string
          kind:
            description: |-
              Kind is a string value representing the REST resource this object represents.
              Servers may infer this from the endpoint the client submits requests to.
              Cannot be updated.
              In CamelCase.
              More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
            type: string
          metadata:
            type: object
          spec:
            description: E2NodeSetSpec defines the desired state of E2NodeSet
            properties:
              replicas:
                description: Replicas is the number of simulated E2 Nodes to run.
                format: int32
                minimum: 0
                type: integer
            required:
            - replicas
            type: object
          status:
            description: E2NodeSetStatus defines the observed state of E2NodeSet
            properties:
              readyReplicas:
                description: ReadyReplicas is the number of E2 Nodes that are ready.
                format: int32
                type: integer
            type: object
        type: object
    served: true
    storage: true
    subresources:
      status: {}
status:
  acceptedNames:
    kind: E2NodeSet
    listKind: E2NodeSetList
    plural: e2nodesets
    singular: e2nodeset
  conditions:
  - lastTransitionTime: "2025-07-26T17:48:45Z"
    message: no conflicts found
    reason: NoConflicts
    status: "True"
    type: NamesAccepted
  - lastTransitionTime: "2025-07-26T17:48:45Z"
    message: the initial names have been accepted
    reason: InitialNamesAccepted
    status: "True"
    type: Established
  storedVersions:
  - v1alpha1
```

### 2. Output of `kubectl api-resources | grep e2nodeset`
*(This is the key test. An empty output here indicates the API server has not registered the CRD.)*
```
e2nodesets                                       nephoran.com/v1alpha1               true         E2NodeSet
```

### 3. Output of `kubectl get pods -n kube-system`
*(This shows the health of the core control plane components.)*
```
NAME                                      READY   STATUS      RESTARTS        AGE
coredns-697968c856-r2r49                  1/1     Running     3 (6h43m ago)   57d
helm-install-traefik-crd-lnqhp            0/1     Completed   0               57d
helm-install-traefik-zwghh                0/1     Completed   1               57d
local-path-provisioner-774c6665dc-nxvxf   1/1     Running     3 (6h43m ago)   57d
metrics-server-6f4c6675d5-dc8sh           1/1     Running     3 (6h43m ago)   57d
svclb-nephio-webui-3c9d60df-bf7rw         1/1     Running     0               135m
svclb-traefik-18ee8db1-z47k8              2/2     Running     6 (6h43m ago)   57d
traefik-c98fdf6fb-pbxq8                   1/1     Running     3 (6h43m ago)   57d
```

### 4. Output of `kubectl logs -n kube-system -l component=kube-apiserver --tail=100`
*(These are the logs from the API server itself, which may contain relevant errors.)*
```
[PASTE OUTPUT HERE]
```

### 5. Output of `kubectl version`
*(Provides context on the cluster and client versions.)*
```
[PASTE OUTPUT HERE]
```

### 6. Output of `kubectl get --raw /readyz?verbose`
*(This shows the detailed health status of the API server.)*
```
[PASTE OUTPUT HERE]
```

## Suspected Cause

Based on the evidence, we suspect this is a cluster-level issue with the Kubernetes API server. The CRD is successfully written to `etcd` (as confirmed by `kubectl get crd`), but the API server is failing to properly register it and make it available as a new resource type.

Possible causes include an API server caching problem, an issue with the CRD controller within the control plane, or a more general misconfiguration.

Please investigate the health and configuration of the Kubernetes control plane to resolve this issue.

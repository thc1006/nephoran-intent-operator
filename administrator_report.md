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
[PASTE OUTPUT HERE]
```

### 2. Output of `kubectl api-resources | grep e2nodeset`
*(This is the key test. An empty output here indicates the API server has not registered the CRD.)*
```
[PASTE OUTPUT HERE]
```

### 3. Output of `kubectl get pods -n kube-system`
*(This shows the health of the core control plane components.)*
```
[PASTE OUTPUT HERE]
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

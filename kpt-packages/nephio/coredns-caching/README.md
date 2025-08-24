# coredns-caching

## Description
CoreDNS application configured for the caching layer.

## Usage

This is a simple example package, requiring minimal customization for
deployment. It creates a `Deployment`, `Service`, and `ConfigMap`. The
`ConfigMap` is not intended to be customized, it sets up CoreDNS as a simple
caching-only DNS server.

The namespace for the resources will be set based upon the `name` field in the
`package-context.yaml` during the render pipeline. When using `kpt pkg get` with
`--for-deployment` this value is automatically set to the local directory name.
Similarly, when using `kpt alpha rpkg clone` or the Porch UI, this value is set
to the `packageName` of the `PackageRevision`.

Note that the `Namespace` resource itself is not created by this package, and
must be provisioned another way.

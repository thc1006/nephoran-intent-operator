# coredns-caching

## Description
CoreDNS application configured for the caching layer, with scaling profile.

## Usage

This is an example of defining an API for configuring a package that is not
package-specific, but does represent a common set of customizations made on
packages.

In this case, we have taken the original coredns-caching package and added a
Nephio `ClusterScaleProfile` resource, along with functions that operate on that
resource. This resource represents the interface by which the Nephio automation
controller can modify this package for the particular cluster in which it is
being deployed. It is important to design these resources and the functions that
accept them as input, such that they can be used across many packages.

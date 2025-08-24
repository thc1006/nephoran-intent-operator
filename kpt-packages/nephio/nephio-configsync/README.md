# nephio-configsync

## Description
Package for the ConfigSync on Nephio workload clusters.

To use this package in your own environment, you should modify the `repo` field
in the `rootsync.yaml` file to point to your GitHub user. The repo name itself
will be changed during `kpt fn render` to point to the repo with the same name
as the package instance.

Thus, you could clone this package from the upstream Nephio repository, and
modify the `repo` value, then store it in your organizational repository and
consume it from there in the future. This would eliminate any need to customize
package on each instantiation. If you are using a different type of repository
with a different `repo` URL format, you may also have to adjust the
configuration of in `apply-replacements.yaml`.

This package is also configured for no Git authentication or authorization. See
the Config Sync
[documentation](https://cloud.google.com/anthos-config-management/docs/how-to/installing-config-sync#git-creds-secret)
for details.

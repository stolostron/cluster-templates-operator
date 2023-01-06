# Dev guide
## Deploying the latest version
To deploy the latest vesion of the CLaaS operator, run the following commands:
- `export QUAY_USERNAME=<your_quay_username>`
- `make bundle bundle-build bundle-push BUNDLE_IMG="quay.io/$QUAY_USERNAME/cluster-templates-operator-bundle"`
- make sure that the repo `quay.io/$QUAY_USERNAME/cluster-templates-operator-bundle` is public
- `operator-sdk run bundle quay.io/$QUAY_USERNAME/cluster-templates-operator-bundle:latest --timeout 5m`

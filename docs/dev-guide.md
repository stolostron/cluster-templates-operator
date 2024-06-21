# Developer guide
## Deploying the latest version
To deploy the latest vesion of the CLaaS operator, run the following commands:
- `export QUAY_USERNAME=<your_quay_username>`
- `make bundle bundle-build bundle-push BUNDLE_IMG="quay.io/$QUAY_USERNAME/cluster-templates-operator-bundle"`
- make sure that the repo `quay.io/$QUAY_USERNAME/cluster-templates-operator-bundle` is public
- `operator-sdk run bundle quay.io/$QUAY_USERNAME/cluster-templates-operator-bundle:latest --timeout 5m`

# Releasing a new version to OperatorHub

CaaS is being released to [K8s Community Operators](https://github.com/k8s-operatorhub/community-operators) and [OpenShift Community Operators](https://github.com/redhat-openshift-ecosystem/community-operators-prod)

## Preparing a release

Run `make manifests && make generate && make bundle` which will generate up-to-date manifests in `bundle` folder.

Update `cluster-aas-operator.clusterserviceversion.yaml` file:
 - update `spec.version` field
 - update `metadata.name` field 
 - update container image in `metadata.annotations.containerImage` and `spec.install.spec.deployments.image`

## Releasing K8s Community Operator

- clone [K8s Community Operators](https://github.com/k8s-operatorhub/community-operators)
- add new folder to `operators/cluster-aas-operator`. Folder name will be a new version name.
- Copy the bundle manifests 
- Since on vanilla k8s environment, `service.beta.openshift.io/serving-cert-secret-name` is not available, the bundle manifests have to be updated:
    - remove `volumeMount`, `volume`, `--tls-cert-file` and `--tls-private-key-file` from `cluster-aas-operator.clusterserviceversion.yaml` spec. Otherwise the controller pod will wait for `secret` to be created and mounted. Which never happens and pod gets stuck.
- Create a PR for the repo.


## Releasing OpenShift Community Operator

- clone [OpenShift Community Operators](https://github.com/redhat-openshift-ecosystem/community-operators-prod)
- add new folder to `operators/cluster-aas-operator`. Folder name will be a new version name.
- Copy the bundle manifests
- Add `com.redhat.openshift.versions: <min_ocp_version>` annotation to `metadata/annotations.yaml`
- Create a PR for the repo.
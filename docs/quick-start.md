# Quick start

`Cluster as a service` creates `ClusterTemplate` named `hypershift-template` by default, if you have Hypershift operator running on your Kubernetes cluster.

The template creates Hypershift-based OpenShift cluster which has no workers - you won't be able to run any workloads on such cluster, but it is fine to explore how `cluster as a service` operator works.

## Template description
Explore the ClusterTemplate definition
`kubectl get ct hypershift-cluster -o yaml`

The result will look like:
```yaml
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplate
metadata:
  name: hypershift-cluster
spec:
  argocdNamespace: argocd
  clusterDefinition:
    destination:
      namespace: clusters
      server: https://kubernetes.default.svc
    project: default
    source:
      chart: hypershift-template
      repoURL: https://stolostron.github.io/cluster-templates-operator
      targetRevision: 0.0.2
    syncPolicy:
      automated: {}
  cost: 1
```

To learn what each of the `spec` fields represent go to [ClusterTemplate CR docs](./cluster-template.md)

In short:
 - `spec.argocdNamespace` defines in which namespace ArgoCD Application will be created. ArgoCD instance has to be running in this namespace. Read more about the ArgoCD setup [here](./argocd.md).
 - `spec.clusterDefinition` defines how a new cluster should look like. In this case a cluster is backed by Helm chart. 

### Helm chart description
Helm chart of `hypershift-cluster` can be found at [Helm chart repository](https://github.com/stolostron/cluster-templates-operator/tree/helm-repo/hypershift-template). It is a typical Helm chart which deploys one resource - `HostedCluster` (Hypershift). The helm chart has 4 properties defined by `values.yaml` and `values.schema.yaml`.

The `HostedCluster` of the Helm chart looks like:
```yaml
apiVersion: hypershift.openshift.io/v1alpha1
kind: HostedCluster
metadata:
  name: {{ .Release.Namespace }}-{{ .Release.Name }}
spec:
  release:
    image: quay.io/openshift-release-dev/ocp-release:{{ .Values.ocpVersion }}-{{ .Values.ocpArch }}
  pullSecret:
    name: pullsecret-cluster
  sshKey:
    name: sshkey-cluster
  networking:
    podCIDR: 10.132.0.0/14
    serviceCIDR: 172.31.0.0/16
    machineCIDR: 192.168.122.0/24
    networkType: OVNKubernetes
  platform:
    type: None
  infraID: {{ .Release.Namespace }}-{{ .Release.Name }}
  dns:
    baseDomain: {{ .Values.baseDnsDomain }}
  services:
  - service: APIServer
    servicePublishingStrategy:
      type: {{ .Values.APIPublishingStrategy }}
  - service: OAuthServer
    servicePublishingStrategy:
      type: Route
  - service: OIDC
    servicePublishingStrategy:
      type: Route
  - service: Konnectivity
    servicePublishingStrategy:
      type: Route
  - service: Ignition
    servicePublishingStrategy:
      type: Route
```

There is a couple of fields to notice:
- `HostedCluster` is a namespaced resource, but we do not have any namespace defined. However `ClusterTemplate` `spec.clusterDefinition.destination.namespace` is set to `clusters` which means that the ArgoCD will set `clusters` namespace to every namespace-scoped resource. So our `HostedCluster` will end up in `clusters` namespace.
- We have a couple of secrets referenced in `HostedCluster`. It is expected that these secrets are already in `clusters` namespace. The secrets are:
  - `spec.pullSecret.name` references `pullsecret-cluster`
  - `spec.sshKey.name` references `sshkey-cluster`

## How to use the template
In order to use the template you need

1. ArgoCD instance running in `argocd` namespace. You can use this sample instance definition:
```yaml
kind: ArgoCD
apiVersion: argoproj.io/v1alpha1
metadata:
  name: argocd-sample
  namespace: argocd
spec:
  controller:
    resources:
      limits:
        cpu: 2000m
        memory: 2048Mi
      requests:
        cpu: 250m
        memory: 1024Mi
  ha:
    enabled: false
    resources:
      limits:
        cpu: 500m
        memory: 256Mi
      requests:
        cpu: 250m
        memory: 128Mi
  redis:
    resources:
      limits:
        cpu: 500m
        memory: 256Mi
      requests:
        cpu: 250m
        memory: 128Mi
  repo:
    resources:
      limits:
        cpu: 1000m
        memory: 512Mi
      requests:
        cpu: 250m
        memory: 256Mi
  server:
    resources:
      limits:
        cpu: 500m
        memory: 256Mi
      requests:
        cpu: 125m
        memory: 128Mi
    route:
      enabled: true
```
2. `clusters` namespace has to be present and managed by ArgoCD
```yaml
kind: Namespace
apiVersion: v1
metadata:
  name: clusters
  labels:
    argocd.argoproj.io/managed-by: argocd
```

3. `clusters` namespace has to contain two secrets - OpenShift pull secret and ssh public key
```yaml
kind: Secret
apiVersion: v1
metadata:
  name: pullsecret-cluster
  namespace: clusters
stringData:
  .dockerconfigjson: '<your_pull_secret>'
type: kubernetes.io/dockerconfigjson
---
apiVersion: v1
kind: Secret
metadata:
  name: sshkey-cluster
  namespace: clusters
stringData:
  id_rsa.pub: <your_public_ssh_key>
```

4. Create `ClusterTemplateQuota` CR which will allow creating instances of `hypershift-cluster` template in quota's namespace
```yaml
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplateQuota
metadata:
  name: example
  namespace: default
spec:
  allowedTemplates:
    - name: hypershift-cluster
```

5. Create `ClusterTemplateInstance`
```yaml
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplateInstance
metadata:
  name: mycluster
  namespace: default
spec:
  clusterTemplateRef: hypershift-cluster
  # we can also pass parameters to Helm chart. For example a custom ocp version. If not specified, the default is taken from Helm chart's values.yaml
  parameters:
    - name: ocpVersion
      value: 4.11.17
```

Once the `status.phase` of ClusterTemplateInstance set to `Ready`, the cluster is ready be used. To access the new cluster, a kubeconfig, admin credentials and API URL are exposed in `status`
 - `status.kubeconfig` - References a secret which contains kubeconfig
 - `status.adminPassword` - References a secret which contains admin credentials
 - `status.apiServerURL` - API URL of a new cluster

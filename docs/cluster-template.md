# ClusterTemplate
ClusterTemplate CR represents a template of a cluster. The template contains both cluster installation and post installation cluster setup.

`Cluster as a service` operator uses abilities of ArgoCD to deploy and manage cluster installation and cluster setup. Therefore it is important to understand `ApplicationSet` CR which is best described in official [ArgoCD docs](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/)

## Example ClusterTemplate
```yaml
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplate
metadata:
  name: my-template
spec:
  clusterDefinition: clusterdefinition
  clusterSetup:
    - clustersetupdefinition
  #Cost of the cluster, used for ClusterTemplateQuota-s
  cost: 1
```

## Example ApplicationSet for cluster installation
```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: clusterdefinition
  namespace: argocd
spec:
  generators:
  - {}
  template:
    spec:
      destination:
        namespace: clusters
        server: '{{ url }}'
      project: default
      source:
        chart: hypershift-template
        repoURL: https://stolostron.github.io/cluster-templates-operator
        targetRevision: 0.0.3
        helm:
          # Fixed parameters (cannot be overridden by the ClusterTemplateInstance)
          parameters:
          - name: ocpVersion
            value: 4.15.0
      syncPolicy:
        automated: {}
```

## Example ApplicationSet for cluster installation
```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: clustersetupdefinition
  namespace: argocd
spec:
  generators:
  - {}
  template:
    spec:
      destination:
        namespace: clusters
        server: '{{ url }}'
      project: default
      source:
        chart: day2installation
        repoURL: https://stolostron.github.io/cluster-templates-operator
        targetRevision: 0.0.3
      syncPolicy:
        automated: {}
```

## Cluster installation definition
Installation of a cluster is defined in `spec.clusterDefinition` field. The content of the field is `name` of ArgoCD `ApplicationSet` CR - see [ArgoCD docs](https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/).

### ApplicationSet source
Any ApplicationSet source can be used - we usually focus on Helm chart source as it allows for easy parameterization of cluster definition yamls, but if you do not need that, feel free to use any other ApplicationSet source.

### ApplicationSet destination
The operator supports deploying clusters to local (hub) cluster only - `destination.server` needs to be set to `https://kubernetes.default.svc`

As a `destination.namespace`, you can use whichever namespace you like (but make sure ArgoCD can sync to it). If you decide to use this field, typically you have 2 options:
  - hardcode namespace value (ie `clusters`) - all namespaced resources will be created in this namespace.
  - If not specified the destionation will be used the same namespace as defined in `ApplicationSet` template specification.

## Cluster setup definition
Post install configuration of a cluster is defined in `spec.clusterSetup`. This field is an array of names of the `ApplicationSet`.

### Application source
Same as with Cluster installation definition, any `ApplicationSet` source can be used.

### ApplicationSet destination
The operator will dynamically set the url of the new cluster once it is available.

## Cluster cost
Every `ClusterTemplate` has a cost defined by `spec.cost` field. The cost is used by `ClusterTemplateQuota`-s to determine wheter a user has enough budget to create a new cluster. More about [ClusterTemplateQuota](./cluster-template-quota.md).

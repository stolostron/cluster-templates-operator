# ClusterTemplate
ClusterTemplate CR represents a template of a cluster. The template contains both cluster installation and post installation cluster setup.

`cluster as a service` operator uses abilities of ArgoCD to deploy and manage cluster installation and cluster setup. Therefore it is important to understand `Application` CR which is best described in official [ArgoCD docs](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#applications)

## Example ClusterTemplate
```yaml
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplate
metadata:
  name: my-template
spec:
  #ns into which Applications (clusterDefinition & clusterSetup) will be created
  argocdNamespace: argocd
  clusterDefinition:
    destination:
      #field will be set dynamically to match ClusterTemplateInstance's namespace
      namespace: ${instance_ns}
      #local (hub) cluster
      server: 'https://kubernetes.default.svc'
    project: ''
    source:
      #source is Helm chart but it can be anything supported by ArgoCD Application
      chart: hypershift-template
      repoURL: 'https://stolostron.github.io/cluster-templates-operator'
      targetRevision: 0.0.2
    #Application sync policy - in this case, sync once
    syncPolicy:
      automated: {}
  clusterSetup:
    - name: day2-setup
      spec:
        destination:
          #field will be set dynamically to match new cluster's API url
          server: #{new_cluster}
        project: '''
        source:
          #any source supported by ArgoCD
          repoURL: 'foo'
        #Application sync policy
        syncPolicy:
          automated: {}
  #Cost of the cluster, used for ClusterTemplateQuota-s
  cost: 1
```

## Application's namespace
`spec.argoCDNamespace` is a required field which tells the operator which namespace should be used to create `Application` CR. It is expected that this namespace is being watched by `ArgoCD` operator. Read more about required [ArgoCD setup](./argocd.md).

## Cluster installation definition
Installation of a cluster is defined in `spec.clusterDefinition` field. The content of the field is `spec` of ArgoCD `Application` CR - see [ArgoCD docs](https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#applications).

### Application source
Any application source can be used - we usually focus on Helm chart source as it allows for easy parameterization of cluster definition yamls, but if you do not need that, feel free to use any other Application source.

### Application destination
The operator supports deploying clusters to local (hub) cluster only - `destination.server` needs to be set to `https://kubernetes.default.svc`

As a `destination.namespace`, you can use whichever namespace you like (but make sure ArgoCD can sync to it). If you decide to use this field, typically you have 2 options:
  - set namespace to `${instance_ns}` - the field will be dynamically set to the namespace of `ClusterTemplateInstance`.
  - hardcode namespace value (ie `clusters`) - all namespaced resources will be created in this namespace.

## Cluster setup definition
Post install configuration of a cluster is defined in `spec.clusterSetup`. This field is an array - every item has a `name` and `spec` (spec of the ArgoCD Application). Cluster setup definition is optional.

### Application source
Same as with Cluster installation definition, any Application source can be used.

### Application destination
As a destination you will typically want to use your new cluster - set the destination to `destination.server: ${new_cluster}`. The operator will dynamically set the url of the new cluster once it is available.
You can also target local (hub) cluster or any other cluster that ArgoCD already recognizes.

## Cluster cost
Every `ClusterTemplate` has a cost defined by `spec.cost` field. The cost is used by `ClusterTemplateQuota`-s to determine wheter a user has enough budget to create a new cluster. More about [ClusterTemplateQuota](./cluster-template-quota.md).
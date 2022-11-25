# Cluster as a service operator
**Self-service clusters with guardrails!** Cluster as a service operator provides an easy way to define clusters as templates and allows creating instances of those templates even for non-privileged developer/devops engineers. Cluster templates operator also allows specifing quotas for the developer/devops engineers.

## Description
Cluster as a service operator adds 3 new CRDs.

**ClusterTemplate** - cluster-scoped resource which defines day1 (cluster installation) and day2 (cluster setup) operations. Both day1 and day2 are defined as argocd Applications. To allow easy customization of the cluster, the argocd Application source is usually helm chart.

**ClusterTemplateQuota** - namespace-scoped resource which defines which ClusterTemplates can be instantiated in a given namespace

**ClusterTemplateInstance** - namespace-scoped resource which represents a request for instance of ClusterTemplate

[Hypershift](https://github.com/openshift/hypershift) and [Hive](https://github.com/openshift/hive) (both ClusterDeployment and ClusterClaim) clusters are supported.

The intended flows for admin and developer/devops engineer

![ClusterTemplates](https://user-images.githubusercontent.com/2078045/193281667-1e1de2ce-9eab-4079-9ab9-f2c0d91a3e50.jpg)


## Getting Started
You’ll need:
1. Kubernetes cluster to run against
2. Hypershift or Hive operator for cluster installation.

### Operator installation
Operator is available on Operator Hub as `Cluster as a service operator`. Once installed, it will pull in ArgoCD operator too (unless it is already available)

### Default template
If you have Hypershift operator installed, the `Cluster as a service operator` will create a default `hypershift-cluster` ClusterTemplate.

Explore the ClusterTemplate definition
`kubectl get ct hypershift-cluster -o yaml`

The result will look like:
```
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
`argocdNamespace` defines namespace where the ArgoCD Application resource will be created

`clusterDefinition` ArgoCD Application spec. In this case a helm chart `hypershift-template` will be deployed by ArgoCD. All resources of this helm chart will be deployed into `clusters` namespace.

`cost` the cost of the cluster. This value is used for the quotas.


### ArgoCD setup
In order to use this template, we need to make sure ArgoCD is setup properly:
1. Since `argocdNamespace` is set to `argocd` we need to ensure that ArgoCD is watching Applications is the namespace - ArgoCD instance needs to be running there. You can use this sample ArgoCD instance:
```
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
```
kind: Namespace
apiVersion: v1
metadata:
  name: clusters
  labels:
    argocd.argoproj.io/managed-by: argocd
```

### Default template prerequisites

1. If you explore helm chart that is being used by the `hypershift-cluster` template you will find out that it expects to find 2 secrets in `clusters` namespace. These secrets contain pull secret and ssh public key
```
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

2. Create `ClusterTemplateQuota` CR which will allow creating instances of `hypershift-cluster` template in quota's namespace
```
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplateQuota
metadata:
  name: example
  namespace: default
spec:
  allowedTemplates:
    - name: hypershift-cluster
```
3. The operator creates 2 ClusterRoles which have the minimal permissions defined for users which will want to self-service their clusters. Lets bind these roles to user `devuser`

  - ClusterRoleBinding gives permissions to `devuser` to read ClusterTemplate-s
  ```
  kind: ClusterRoleBinding
  apiVersion: rbac.authorization.k8s.io/v1
  metadata:
    name: devuser-cluster-templates
  subjects:
    - kind: User
      apiGroup: rbac.authorization.k8s.io
      name: devuser
  roleRef:
    apiGroup: rbac.authorization.k8s.io
    kind: ClusterRole
    name: cluster-templates-user-ct
  ```
  
  - RoleBinding gives permissions to `devuser` to CRUD ClusterTemplateInstance-s and read for ClusterTemplateQuota-s in `devuserns` namespace.

  
  ```
  kind: RoleBinding
  apiVersion: rbac.authorization.k8s.io/v1
  metadata:
    name: devuser-templates
    namespace: devuserns
  subjects:
    - kind: User
      apiGroup: rbac.authorization.k8s.io
      name: devuser
  roleRef:
    apiGroup: rbac.authorization.k8s.io
    kind: ClusterRole
    name: cluster-templates-user
  ```

Now eveything is setup for `devuser` to self-service a cluster.

## Using ClusterTemplate-s as developer/devops engineer
With the setup descibed above, `devuser` can login to hub cluster and:
- Explore ClusterTemplate-s `kubectl get ct`
- Explore ClusterTemplateQuota-s `kubectl get ctq`
- And finally create a new clusters:
```
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplateInstance
metadata:
  name: mycluster
  namespace: devuserns
spec:
  clusterTemplateRef: hypershift-cluster
```

Once the `status.phase` of ClusterTemplateInstance set to `Ready`, the cluster is ready be used. To access the new cluster, a kubeconfig, admin credentials and API URL are exposed in `status`
 - `status.kubeconfig` - References a secret which contains kubeconfig
 - `status.adminPassword` - References a secret which contains admin credentials
 - `status.apiServerURL` - API URL of a new cluster

## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.


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
2. Hypershift and Hive operator for cluster installation.
3. ArgoCD operator

### Running on the cluster
1. Build and push your image to the location specified by `IMG`:
	
```sh
make docker-build docker-push IMG=<some-registry>/cluster-templates-operator:tag
```
	
2. Deploy the operator to the cluster with the image specified by `IMG`:

```sh
operator-sdk run bundle <some-registry>/cluster-templates-operator-bundle:latest
```

### Try It Out
The operator already ships with `hypershift-template` ClusterTemplate.

Explore the ClusterTemplate definition
`kubectl get ct hypershift-template -o yaml`

#### Create new Cluster

1. Create `clusters` namespace and add following secrets
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

2. Create ClusterTemplateQuota CR which will allow creating instances of `hypershift-template` template in quota's namespace
```
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplateQuota
metadata:
  name: example
  namespace: default
spec:
  allowedTemplates:
    - name: hypershift-template
```

3. Finally, create ClusterTemplateInstance

```
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplateInstance
metadata:
  name: mycluster
  namespace: default
spec:
  clusterTemplateRef: hypershift-template
```

Now you will need to wait for the cluster to be ready. Observe the ClusterTemplateInstance's status to get the latest info on the progress.


### Uninstall the operator
To delete the operator from the cluster:

```sh
operator-sdk cleanup cluster-templates-operator
```


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


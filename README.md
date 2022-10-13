# Cluster templates operator
**Self-service clusters with guardrails!** Cluster templates operator provides an easy way to define clusters as templates and allows creating instances of those templates even for non-privileged developer/devops engineers. Cluster templates operator also allows specifing quotas for the developer/devops engineers.

## Description
Cluster templates operator adds 3 new CRDs.

**ClusterTemplate** - cluster-scoped resource which defines how the cluster should look like (using [Helm Chart](https://github.com/helm/helm)) and optionally a post-install configuration of the cluster (using [Tekton Pipeline](https://github.com/tektoncd/pipeline))

**ClusterTemplateQuota** - namespace-scoped resource which defines which ClusterTemplates can be instantiated in a given namespace<br>

**ClusterTemplateInstance** - namespace-scoped resource which represents a request for instance of ClusterTemplate

[Hypershift](https://github.com/openshift/hypershift) and [Hive](https://github.com/openshift/hive) (both ClusterDeployment and ClusterClaim) clusters are supported.

The intended flows for admin and developer/devops enginer

![ClusterTemplates](https://user-images.githubusercontent.com/2078045/193281667-1e1de2ce-9eab-4079-9ab9-f2c0d91a3e50.jpg)


## Getting Started
You’ll need a Kubernetes cluster to run against.

### Running on the cluster
1. Build and push your image to the location specified by `IMG`:
	
```sh
make docker-build docker-push IMG=<some-registry>/cluster-templates-operator:tag
```
	
2. Deploy the operator to the cluster with the image specified by `IMG`:

```sh
operator-sdk run bundle <some-registry>/cluster-templates-operator-bundle:latest
```

### Uninstall the operator
To delete the operator from the cluster:

```sh
operator-sdk cleanup cluster-templates-operator
```

### Try It Out
As mentioned before, every ClusterTemplate is backed by some Helm chart - you can use [this](https://stolostron.github.io/helm-demo/index.yaml) repository as a quickstart where the `hypershift-template` is hosted. The `hypershift-template` is easiest to use (requires Hypershift operator).

1. Create HelmChartRepository CR
```
apiVersion: helm.openshift.io/v1beta1
kind: HelmChartRepository
metadata:
  name: cluster-charts
spec:
  connectionConfig:
    url: 'https://stolostron.github.io/helm-demo/index.yaml'
```

2. Create ClusterTemplate CR which will use the `cluster-charts` HelmChartRepository and `hypershift-template` Helm Chart
```
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplate
metadata:
  name: my-cluster-template
spec:
  cost: 10
  helmChartRef:
    name: hypershift-template
    repository: cluster-charts
    version: 0.1.0
```

3. Create ClusterTemplateQuota CR which will allow 1 instance of `my-cluster-template` template in quota's namespace
```
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplateQuota
metadata:
  name: example
  namespace: default
spec:
  allowedTemplates:
    - name: my-cluster-template
      count: 1
```

4. Finally, create ClusterTemplateInstance

```
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplateInstance
metadata:
  name: mycluster
  namespace: default
spec:
  clusterTemplateRef: my-cluster-template
```

Now you will just need for cluster to be created. Observer the ClusterTemplateInstance's status to get the latest info aby the progress.


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


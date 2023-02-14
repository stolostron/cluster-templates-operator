# Cluster as a service operator
**Self-service clusters with guardrails!** Cluster as a service operator provides an easy way to define clusters as templates and allows creating instances of those templates even for non-privileged developer/devops engineers. CCluster as a service operator also allows specifing quotas for the developer/devops engineers.

## Description
Cluster as a service operator adds 3 new CRDs.

**ClusterTemplate** - cluster-scoped resource which defines day1 (cluster installation) and day2 (cluster setup) operations. Both day1 and day2 are defined as argocd Applications. To allow easy customization of the cluster, the argocd Application source is usually helm chart.

**ClusterTemplateQuota** - namespace-scoped resource which defines which ClusterTemplates can be instantiated in a given namespace

**ClusterTemplateInstance** - namespace-scoped resource which represents a request for instance of ClusterTemplate

[Hypershift](https://github.com/openshift/hypershift) and [Hive](https://github.com/openshift/hive) (both ClusterDeployment and ClusterClaim) clusters are supported.

The intended flows for admin and developer/devops engineer

![ClusterTemplates](https://user-images.githubusercontent.com/2078045/204266251-53a60909-648d-439a-b085-00b7d6bc0f17.jpg)


## Prerequisities
You’ll need:
1. Kubernetes cluster to run against
2. [Hypershift](https://github.com/openshift/hypershift) or [Hive](https://github.com/openshift/hive) operator for cluster installation.

## Operator installation
Operator is available on Operator Hub as `Cluster as a service operator`. Once installed, it will pull in ArgoCD operator too (unless it is already available). 

## Documentation

[Getting started guide](./docs/quick-start.md)

[Full docs](./docs/index.md)


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


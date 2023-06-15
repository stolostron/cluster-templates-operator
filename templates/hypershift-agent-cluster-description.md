# Hypershift Cluster with Agent Based Workers

## Description
A hypershift template with worker nodes taken from an infrastructure environment managed by the infrastructure operator. It has one worker node only, but is capable of running real workload.

## Features
- The spoke cluster installed from this template is fully co-located on this hub cluster

## Prerequisites
- Enable hypershift ([docs](https://access.redhat.com/documentation/en-us/red_hat_advanced_cluster_management_for_kubernetes/2.7/html-single/clusters/index#hosted-enable-feature-aws))
- Create an infrastructure environment called `agent-infra` (in UI go to `All Clusters` -> `Host Inventory` -> `Create infrastructure environment`). Once created, add/discover at least one host.

- Create a `clusters` namespace: 
```yaml
kind: Namespace
apiVersion: v1
metadata:
  name: clusters
  labels:
    argocd.argoproj.io/managed-by: argocd
```
- Create 2 secrets - one which contains the [pull-secret](https://console.redhat.com/openshift/install/pull-secret) and another one for the ssh public key
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

## Usage
Create an instance by creating the following yaml:

```yaml
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplateInstance
metadata:
  name: hsclskubevirt
  namespace: clusters
spec:
  clusterTemplateRef: hypershift-agent-cluster
```

## Support
If you hit a problem, please report an [issue](https://github.com/stolostron/cluster-templates-manifests/issues)
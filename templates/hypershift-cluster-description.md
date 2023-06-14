# Hypershift Cluster (not for production use!)

## Description

A hypershift template with NO WORKER nodes. No workload can be run on clusters created from this template.
This template is suitable for learning how to configure and use cluster templates end to end.

## Features

- Only installs control plane
- Does not need any external or internal provider for running the workload

## Prerequisites

- Enable hypershift ([docs](https://access.redhat.com/documentation/en-us/red_hat_advanced_cluster_management_for_kubernetes/2.7/html-single/clusters/index#hosted-enable-feature-aws))

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
  name: hsclsempty
  namespace: clusters
spec:
  clusterTemplateRef: hypershift-cluster
```

## Support

If you hit a problem, please report an [issue](https://github.com/stolostron/cluster-templates-manifests/issues)
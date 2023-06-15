# Hypershift Kubevirt Cluster

## Description

A template which installs clusters with hypershift control plane and workers on kubevirt virtual machines running on the same cluster.

## Features

- The spoke cluster installed from this template is fully co-located on this hub cluster

## Prerequisites

- Enable hypershift ([docs](https://access.redhat.com/documentation/en-us/red_hat_advanced_cluster_management_for_kubernetes/2.7/html-single/clusters/index#hosted-enable-feature-aws))
- Install OpenShift Virtualization on to this cluster from [operatorhub](operatorhub/all-namespaces?keyword=virtualization&details-item=kubevirt-hyperconverged-redhat-operators-openshift-marketplace)
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
- If you are using the UI, continue by following the Getting started on top of the page.

- If you are not using the UI, create an instance by creating the following yaml:

```yaml
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplateInstance
metadata:
  name: hsclskubevirt
  namespace: clusters
spec:
  clusterTemplateRef: hypershift-kubevirt-cluster
```

## Support

If you hit a problem, please report an [issue](https://github.com/stolostron/cluster-templates-manifests/issues)
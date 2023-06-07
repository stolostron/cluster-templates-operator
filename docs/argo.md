# Configuring ArgoCD
ArgoCD is installed by default as a dependency of the CaaS operator. But it's not configured to be used by the operator.
This document provide information how to setup basic ArgoCD instance, which will work with CaaS operator.

1. CaaS operator use `argocd` namespace by default. We need to create it:
   ```oc create ns argocd```

2. Create the ArgoCD instance
```yaml
apiVersion: argoproj.io/v1alpha1
kind: ArgoCD
metadata:
  name: example-argocd
  namespace: argocd
spec: {}
```

3. Patch subscription of the ArgoCD which will setup Argo as cluster-scoped

  a. If you installed CaaS operator via OperatorHub you should have installed upstream ArgoCD
     as a dependency, then patch following subscription:

```bash
$ oc edit subscriptions.operators.coreos.com -n openshift-operators $(oc get subscriptions.operators.coreos.com -n openshift-operators -l operators.coreos.com/argocd-operator.openshift-operators="" -o=jsonpath='{.items[0].metadata.name}')
```

And add following configuration to spec of the Subscription

```yaml
config:
  env:
    - name: ARGOCD_CLUSTER_CONFIG_NAMESPACES
      value: argocd
```

  b. If you used `openshift-gitops` before the CaaS operator was installed, upstream Argo was not installed, in
     that case you need to update `openshift-gitops` subscription as follows:

```bash
$ oc edit subscriptions.operators.coreos.com -n openshift-operators $(oc get subscriptions.operators.coreos.com -n openshift-operators -l operators.coreos.com/openshift-gitops-operator.openshift-operators="" -o=jsonpath='{.items[0].metadata.name}')
```

And add following configuration to spec of the Subscription

```yaml
config:
  env:
    - name: ARGOCD_CLUSTER_CONFIG_NAMESPACES
      value: argocd
```

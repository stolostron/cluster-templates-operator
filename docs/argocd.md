# ArgoCD
`cluster as a service` operator uses abilities of ArgoCD to deploy and manage cluster installation and cluster setup. Therefore it is necessary to properly setup ArgoCD.

## ArgoCD Instance
Every `ClusterTemplate` has `spec.argoCDNamespace` field. The field defines in which namespace an ArgoCD Application will be created. It is expected that the Application will be picked up by ArgoCD and acted upon. 

ArgoCD is watching `Application`-s only in namespace of the ArgoCD instance.

The default template `hypershift-template` has `spec.argoCDNamespace` set to `argocd` - in order to use this template, you need to make sure that ArgoCD instance is running in this namespace.

An example ArgoCD instance definition looks like:
```yaml
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
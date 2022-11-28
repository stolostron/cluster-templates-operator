# ClusterTemplateQuota
`ClusterTemplateQuota` CR is a namespaced resource that specifies which templates can be used in a given namespace. It is a similar concept to build-in [k8s resource quotas](https://kubernetes.io/docs/concepts/policy/resource-quotas/) but focusing on `ClusterTemplate`-s.

A `ClusterTemplateQuota` looks like:
```yaml
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplateQuota
metadata:
  name: my-quota
  namespace: my-namespace
spec:
  allowedTemplates:
    - name: aws-small
      count: 5
    - name: aws-large
      count: 2
  budget: 50
```

This quota allows creating 5 instances of `aws-small` and 2 instances of `aws-large` in `my-namespace` namespace. All clusters also needs to fit within the budget (50).

For example if `aws-small` template has `spec.const` set to `10` and `aws-large` has cost `25`, users will be able to create:
 - 10 instances of `aws-small` or,
 - 2 instances of `aws-large` or,
 - 1 instance of `aws-large` and 2 instances of `aws-small` (any more would go over assigned budget)

Both `spec.count` and `spec.budget` are optional. If you do not want to restrict the amount of the templates, do not specify these fields.

```yaml
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplateQuota
metadata:
  name: my-quota
  namespace: my-namespace
spec:
  allowedTemplates:
    - name: aws-small
    - name: aws-large
```

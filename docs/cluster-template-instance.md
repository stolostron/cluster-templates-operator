# ClusterTemplateInstance
`ClusterTemplateInstance` CR is a namespaced resource which represents an instance of some `ClusterTemplate`.

```yaml
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplateInstance
metadata:
  name: my-cluster
  namespace: my-namespace
spec:
  clusterTemplateRef: aws-small
```

Every `ClusterTemplateInstance` references some `ClusterTemplate` via `spec.clusterTemplateRef` field. In the example above, `aws-small` template is used.

If the referenced `ClusterTemplate` is using Helm chart, we can pass parameters via `spec.parameters` field.

```yaml
apiVersion: clustertemplate.openshift.io/v1alpha1
kind: ClusterTemplateInstance
metadata:
  name: my-cluster
  namespace: my-namespace
spec:
  clusterTemplateRef: aws-small
  parameters:
    # set 'param1' to 'foo' of cluster definition Helm chart
    - name: param1
      value: foo
    # set 'param2' to 'bar' of cluster setup 'day2-setup' Helm chart
    - name: param2
      value: bar
      clusterSetup: day2-setup
```

Once the `ClusterTemplateInstance` is created, you can observe `status.phase` field to see the progress of the cluster creation. Then the cluster is ready, following fields will be populated:
 - `status.kubeconfig` - reference to a secret which contains kubeconfig
 - `status.adminPassword` - reference to a secret which contains admin credentials
 - `status.apiServerURL` - API server URL of a new cluster

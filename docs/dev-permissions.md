# Minimal set of permissions for ClusterTemplateInstance users

Minimal permissions on the hub cluster are:
 - Access to some namespace.
 - `Read` for `ClusterTemplate` - so users can explore the template (ie description) or the status which contains Helm chart's values and values schema if there are any.
 - `Read` for `ClusterTemplateQuota` - so users can understand which templates can be used & what is the available budget
 - `CRUD` permissions for `ClusterTemplateInstance`
 - `Read` permissions for secrets of `ClusterTemplateInstance` (kubeconfig and admin credentials)

## Default ClusterRoles
The `cluster as a service` operator creates two ClusterRole-s by default:
 - `cluster-templates-user` - You can use this ClusterRole to give permissions to  `ClusterTemplateInstance` and `ClusterTemplateQuota`. Following RoleBiding adds the permissions to `foo-user` in `devclusters` namespace:
  ```yaml
  kind: RoleBinding
  apiVersion: rbac.authorization.k8s.io/v1
  metadata:
    name: cluster-templates-user-rb
    namespace: devclusters
  subjects:
    - kind: User
      apiGroup: rbac.authorization.k8s.io
      name: foo-user
  roleRef:
    apiGroup: rbac.authorization.k8s.io
    kind: ClusterRole
    name: cluster-templates-user

  ```
 - `cluster-templates-user-ct` - You can use this ClusterRole to give permissions for `ClusterTemplate`. Since `ClusterTemplate` is cluster-scoped resource, you need to create ClusterRoleBinding. Following ClusterRoleBiding adds the permissions to `foo-user`:
  ```yaml
  kind: ClusterRoleBinding
  apiVersion: rbac.authorization.k8s.io/v1
  metadata:
    name: cluster-templates-user-crb
  subjects:
    - kind: User
      apiGroup: rbac.authorization.k8s.io
      name: foo-user
  roleRef:
    apiGroup: rbac.authorization.k8s.io
    kind: ClusterRole
    name: cluster-templates-user-ct
  ```

## Dynamic permissions for ClusterTemplateInstance secrets
When a new cluster is created (via `ClusterTemplateInstance`), the operator will dynamically create Role and RoleBinding to any user that is bound to the `cluster-templates-role`, giving the user access only to secrets referenced by the new cluster (kubeconfig and admin credentials). When `ClusterTemplateInstance` is deleted, the dynamically created Role and RoleBiding are deleted too.



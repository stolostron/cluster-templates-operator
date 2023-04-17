# API Reference

Packages:

- [clustertemplate.openshift.io/v1alpha1](#clustertemplateopenshiftiov1alpha1)

# clustertemplate.openshift.io/v1alpha1

Resource Types:

- [ClusterTemplateInstance](#clustertemplateinstance)

- [ClusterTemplateQuota](#clustertemplatequota)

- [ClusterTemplate](#clustertemplate)




## ClusterTemplateInstance
<sup><sup>[↩ Parent](#clustertemplateopenshiftiov1alpha1 )</sup></sup>






Represents instance of a cluster

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>clustertemplate.openshift.io/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>ClusterTemplateInstance</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#clustertemplateinstancespec">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clustertemplateinstancestatus">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterTemplateInstance.spec
<sup><sup>[↩ Parent](#clustertemplateinstance)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>clusterTemplateRef</b></td>
        <td>string</td>
        <td>
          A reference to ClusterTemplate which will be used for installing and setting up the cluster<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#clustertemplateinstancespecparametersindex">parameters</a></b></td>
        <td>[]object</td>
        <td>
          Helm parameters to be passed to cluster installation or setup<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterTemplateInstance.spec.parameters[index]
<sup><sup>[↩ Parent](#clustertemplateinstancespec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the Helm parameter<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value of the Helm parameter<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>clusterSetup</b></td>
        <td>string</td>
        <td>
          Name of the application set to which parameter is applied<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterTemplateInstance.status
<sup><sup>[↩ Parent](#clustertemplateinstance)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#clustertemplateinstancestatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          Resource conditions<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          Additional message for Phase<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>phase</b></td>
        <td>string</td>
        <td>
          Represents instance installaton & setup phase<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#clustertemplateinstancestatusadminpassword">adminPassword</a></b></td>
        <td>object</td>
        <td>
          A reference for secret which contains username and password under keys "username" and "password"<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>apiServerURL</b></td>
        <td>string</td>
        <td>
          API server URL of the new cluster<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clustertemplateinstancestatusclustersetupindex">clusterSetup</a></b></td>
        <td>[]object</td>
        <td>
          Status of each cluster setup<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>clusterTemplateLabels</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clustertemplateinstancestatusclustertemplatespec">clusterTemplateSpec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clustertemplateinstancestatuskubeconfig">kubeconfig</a></b></td>
        <td>object</td>
        <td>
          A reference for secret which contains kubeconfig under key "kubeconfig"<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterTemplateInstance.status.conditions[index]
<sup><sup>[↩ Parent](#clustertemplateinstancestatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource. --- This struct is intended for direct use as an array at the field path .status.conditions.  For example, type FooStatus struct{ // Represents the observations of a foo's current state. // Known .status.conditions.type are: "Available", "Progressing", and "Degraded" // +patchMergeKey=type // +patchStrategy=merge // +listType=map // +listMapKey=type Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"` 
 // other fields }

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another. This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition. This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition. Producers of specific condition types may define expected values and meanings for this field, and whether the values are considered a guaranteed API. The value should be a CamelCase string. This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase. --- Many .condition.type values are consistent across resources like Available, but because arbitrary conditions can be useful (see .node.status.conditions), the ability to deconflict is important. The regex it matches is (dns1123SubdomainFmt/)?(qualifiedNameFmt)<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon. For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterTemplateInstance.status.adminPassword
<sup><sup>[↩ Parent](#clustertemplateinstancestatus)</sup></sup>



A reference for secret which contains username and password under keys "username" and "password"

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterTemplateInstance.status.clusterSetup[index]
<sup><sup>[↩ Parent](#clustertemplateinstancestatus)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          Description of the cluster setup status<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the cluster setup<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>string</td>
        <td>
          Status of the cluster setup<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### ClusterTemplateInstance.status.clusterTemplateSpec
<sup><sup>[↩ Parent](#clustertemplateinstancestatus)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>clusterDefinition</b></td>
        <td>string</td>
        <td>
          ArgoCD applicationset name which is used for installation of the cluster<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>clusterSetup</b></td>
        <td>[]string</td>
        <td>
          Array of ArgoCD applicationset names which are used for post installation setup of the cluster<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>cost</b></td>
        <td>integer</td>
        <td>
          Cost of the cluster, used for quotas<br/>
          <br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>skipClusterRegistration</b></td>
        <td>boolean</td>
        <td>
          Skip the registeration of the cluster to the hub cluster<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterTemplateInstance.status.kubeconfig
<sup><sup>[↩ Parent](#clustertemplateinstancestatus)</sup></sup>



A reference for secret which contains kubeconfig under key "kubeconfig"

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names TODO: Add other useful fields. apiVersion, kind, uid?<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## ClusterTemplateQuota
<sup><sup>[↩ Parent](#clustertemplateopenshiftiov1alpha1 )</sup></sup>






Defines which ClusterTemplates can be used in a given namespace

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>clustertemplate.openshift.io/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>ClusterTemplateQuota</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#clustertemplatequotaspec">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clustertemplatequotastatus">status</a></b></td>
        <td>object</td>
        <td>
          ClusterTemplateQuotaStatus defines the observed state of ClusterTemplateQuota<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterTemplateQuota.spec
<sup><sup>[↩ Parent](#clustertemplatequota)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#clustertemplatequotaspecallowedtemplatesindex">allowedTemplates</a></b></td>
        <td>[]object</td>
        <td>
          Represents all ClusterTemplates which can be used in given namespace<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>budget</b></td>
        <td>integer</td>
        <td>
          Total budget for all clusters within given namespace<br/>
          <br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterTemplateQuota.spec.allowedTemplates[index]
<sup><sup>[↩ Parent](#clustertemplatequotaspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the ClusterTemplate<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>count</b></td>
        <td>integer</td>
        <td>
          Defines how many instances of the ClusterTemplate can exist<br/>
          <br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>deleteAfter</b></td>
        <td>string</td>
        <td>
          Template instance will be removed after specified time This is a Duration value; see https://pkg.go.dev/time#ParseDuration for accepted formats. Note: due to discrepancies in validation vs parsing, we use a Pattern instead of `Format=duration`. See https://bugzilla.redhat.com/show_bug.cgi?id=2050332 https://github.com/kubernetes/apimachinery/issues/131 https://github.com/kubernetes/apiextensions-apiserver/issues/56<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterTemplateQuota.status
<sup><sup>[↩ Parent](#clustertemplatequota)</sup></sup>



ClusterTemplateQuotaStatus defines the observed state of ClusterTemplateQuota

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>budgetSpent</b></td>
        <td>integer</td>
        <td>
          How much budget is currenly spent<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#clustertemplatequotastatustemplateinstancesindex">templateInstances</a></b></td>
        <td>[]object</td>
        <td>
          Which instances are in use<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### ClusterTemplateQuota.status.templateInstances[index]
<sup><sup>[↩ Parent](#clustertemplatequotastatus)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the ClusterTemplate<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>count</b></td>
        <td>integer</td>
        <td>
          Defines how many instances of the ClusterTemplate can exist<br/>
          <br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>deleteAfter</b></td>
        <td>string</td>
        <td>
          Template instance will be removed after specified time This is a Duration value; see https://pkg.go.dev/time#ParseDuration for accepted formats. Note: due to discrepancies in validation vs parsing, we use a Pattern instead of `Format=duration`. See https://bugzilla.redhat.com/show_bug.cgi?id=2050332 https://github.com/kubernetes/apimachinery/issues/131 https://github.com/kubernetes/apiextensions-apiserver/issues/56<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## ClusterTemplate
<sup><sup>[↩ Parent](#clustertemplateopenshiftiov1alpha1 )</sup></sup>






Template of a cluster - both installation and post-install setup are defined as ArgoCD application spec. Any application source is supported - typically a Helm chart

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>clustertemplate.openshift.io/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>ClusterTemplate</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#clustertemplatespec">spec</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#clustertemplatestatus">status</a></b></td>
        <td>object</td>
        <td>
          ClusterTemplateStatus defines the observed state of ClusterTemplate<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterTemplate.spec
<sup><sup>[↩ Parent](#clustertemplate)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>clusterDefinition</b></td>
        <td>string</td>
        <td>
          ArgoCD applicationset name which is used for installation of the cluster<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>clusterSetup</b></td>
        <td>[]string</td>
        <td>
          Array of ArgoCD applicationset names which are used for post installation setup of the cluster<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>cost</b></td>
        <td>integer</td>
        <td>
          Cost of the cluster, used for quotas<br/>
          <br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>skipClusterRegistration</b></td>
        <td>boolean</td>
        <td>
          Skip the registeration of the cluster to the hub cluster<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterTemplate.status
<sup><sup>[↩ Parent](#clustertemplate)</sup></sup>



ClusterTemplateStatus defines the observed state of ClusterTemplate

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#clustertemplatestatusclusterdefinition">clusterDefinition</a></b></td>
        <td>object</td>
        <td>
          Describes helm chart properties and their schema<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#clustertemplatestatusclustersetupindex">clusterSetup</a></b></td>
        <td>[]object</td>
        <td>
          Describes helm chart properties and schema for every cluster setup step<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterTemplate.status.clusterDefinition
<sup><sup>[↩ Parent](#clustertemplatestatus)</sup></sup>



Describes helm chart properties and their schema

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>error</b></td>
        <td>string</td>
        <td>
          Contain information about failure during fetching helm chart<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>schema</b></td>
        <td>string</td>
        <td>
          Content of helm chart values.schema.json<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>string</td>
        <td>
          Content of helm chart values.yaml<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ClusterTemplate.status.clusterSetup[index]
<sup><sup>[↩ Parent](#clustertemplatestatus)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the cluster setup step<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>error</b></td>
        <td>string</td>
        <td>
          Contain information about failure during fetching helm chart<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>schema</b></td>
        <td>string</td>
        <td>
          Content of helm chart values.schema.json<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>string</td>
        <td>
          Content of helm chart values.yaml<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>
# Custom configuration
CaaS can be further customized using the Config CR **config.clustertemplate.openshift.io** called **config**. You can edit it by: `oc edit config.clustertemplate.openshift.io config`.

The following configurations are available:
 - argoCDNamespace: The name of the namespace in which the argocd is running. Default: cluster-aas-operator. **Please note**: after changing this namespace, you have to restart the claas operator (called cluster-aas-operator-controller-manager).
 - uiEnabled: If true, the UI will be automatically installed. Default: true
 - uiImage: A link to a repository containing the image of the UI. Default: depends on the version

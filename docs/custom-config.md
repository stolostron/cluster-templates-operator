# Custom configuration
CaaS can be further customized using the following config map with name **claas-config** in a namespace **cluster-aas-operator**.
The configurations available for this config map are:
 - argocd-ns: The name of the namespace in which the argocd is running. Default: argocd
 - enable-ui: If true, the UI will be automatically installed. Default: false
 - ui-image: A link to a repository containing the image of the UI. Default: quay.io/stolostron/cluster-templates-console-plugin:latest

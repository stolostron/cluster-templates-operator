package controllers

var ArgoCDNamespace = "argocd"

//TODO add controller to allow changing argoCD namespace via configmap
//ns cluster-aas-operator-system
//name claas-config
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

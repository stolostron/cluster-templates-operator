package controllers

import (
	"context"
	"fmt"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	applicationset "github.com/argoproj/applicationset/pkg/utils"
	consoleV1 "github.com/openshift/api/console/v1"
	console "github.com/openshift/api/console/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

type ConsolePluginReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	pluginResourceName = "claas-console-plugin"
	pluginNamespace    = "cluster-aas-operator"
)

var (
	pluginLabels = map[string]string{
		"clustertemplates.openshift.io/component": "console-plugin",
	}
)

func getQuickStarts() []*consoleV1.ConsoleQuickStart {
	return []*consoleV1.ConsoleQuickStart{GetTemplateQuickStart(), GetQuotaQuickStart(), GetShareTemplateQuickStart()}
}

func getQuickStartClientObjects() []client.Object {
	quickStarts := getQuickStarts()
	objects := make([]client.Object, len(quickStarts))
	for i, c := range quickStarts {
		objects[i] = (client.Object)(c)
	}
	return objects
}

func (r *ConsolePluginReconciler) createOrUpdateQuickStarts(
	ctx context.Context,
	req ctrl.Request,
) error {
	quickStarts := getQuickStarts()
	for _, qs := range quickStarts {
		originalQs := &consoleV1.ConsoleQuickStart{
			ObjectMeta: qs.ObjectMeta,
		}
		_, err := applicationset.CreateOrUpdate(ctx, r.Client, originalQs, func() error {
			if !reflect.DeepEqual(originalQs.Spec, qs.Spec) {
				originalQs.Spec = qs.Spec
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func getConsolePlugin() *console.ConsolePlugin {
	return &console.ConsolePlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: "clustertemplates-plugin",
		},
		Spec: console.ConsolePluginSpec{
			DisplayName: "Cluster as a Service plugin",
			Service: console.ConsolePluginService{
				BasePath:  "/",
				Name:      pluginResourceName,
				Namespace: pluginNamespace,
				Port:      9443,
			},
			Proxy: []console.ConsolePluginProxy{
				{
					Type:      "Service",
					Alias:     "repositories",
					Authorize: true,
					Service: console.ConsolePluginProxyServiceConfig{
						Name:      "cluster-aas-operator-repo-bridge-service",
						Namespace: pluginNamespace,
						Port:      8001,
					},
				},
			},
		},
	}
}

func GetPluginDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pluginResourceName,
			Namespace: pluginNamespace,
			Labels:    pluginLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(2),
			Selector: &metav1.LabelSelector{MatchLabels: pluginLabels},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: pluginLabels,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  pluginResourceName,
							Image: UIImage,
							Ports: []v1.ContainerPort{
								{
									ContainerPort: 9443,
									Protocol:      v1.ProtocolTCP,
								},
							},
							ImagePullPolicy: v1.PullAlways,
							SecurityContext: &v1.SecurityContext{
								Capabilities: &v1.Capabilities{
									Drop: []v1.Capability{"ALL"},
								},
								AllowPrivilegeEscalation: pointer.Bool(false),
							},
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceCPU:    *resource.NewMilliQuantity(10, resource.DecimalSI),
									v1.ResourceMemory: resource.MustParse("50Mi"),
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "plugin-serving-cert",
									ReadOnly:  true,
									MountPath: "/var/serving-cert",
								},
								{
									Name:      "nginx-conf",
									ReadOnly:  true,
									MountPath: "/etc/nginx/nginx.conf",
									SubPath:   "nginx.conf",
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "plugin-serving-cert",
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName:  "plugin-serving-cert",
									DefaultMode: pointer.Int32(420),
								},
							},
						},
						{
							Name: "nginx-conf",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: pluginResourceName,
									},
									DefaultMode: pointer.Int32(420),
								},
							},
						},
					},
					RestartPolicy: v1.RestartPolicyAlways,
					DNSPolicy:     v1.DNSClusterFirst,
					SecurityContext: &v1.PodSecurityContext{
						RunAsNonRoot: pointer.Bool(true),
						SeccompProfile: &v1.SeccompProfile{
							Type: v1.SeccompProfileTypeRuntimeDefault,
						},
					},
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "25%",
					},
					MaxSurge: &intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "25%",
					},
				},
			},
		},
	}
}

func getPluginService() *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pluginResourceName,
			Namespace: pluginNamespace,
			Annotations: map[string]string{
				"service.alpha.openshift.io/serving-cert-secret-name": "plugin-serving-cert",
			},
			Labels: pluginLabels,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:     "9443-tcp",
					Protocol: v1.ProtocolTCP,
					Port:     *pointer.Int32(9443),
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 9443,
					},
				},
			},
			Selector:        pluginLabels,
			Type:            v1.ServiceTypeClusterIP,
			SessionAffinity: v1.ServiceAffinityNone,
		},
	}
}

func getPluginCM() *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pluginResourceName,
			Namespace: pluginNamespace,
			Labels:    pluginLabels,
		},
		Data: map[string]string{
			"nginx.conf": `error_log /dev/stdout info;
events {}
http {
  access_log         /dev/stdout;
  include            /etc/nginx/mime.types;
  default_type       application/octet-stream;
  keepalive_timeout  65;
  server {
    listen              9443 ssl;
    ssl_certificate     /var/serving-cert/tls.crt;
    ssl_certificate_key /var/serving-cert/tls.key;
    root                /usr/share/nginx/html;
    location = /plugin-entry.js {
      root   /usr/share/nginx/html;
      expires -1;
      add_header 'Cache-Control' 'no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0';
    }
    location = /plugin-manifest.json {
        root   /usr/share/nginx/html;
        expires -1;
        add_header 'Cache-Control' 'no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0';
    }
  }
}`,
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConsolePluginReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// A channel is used to generate an initial sync event.
	// Afterwards, the controller syncs on the plugin resources.
	initialSync := make(chan event.GenericEvent)
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}, builder.WithPredicates(predicate.NewPredicateFuncs(r.selectPluginDeployment))).
		Watches(&source.Channel{Source: initialSync}, &handler.EnqueueRequestForObject{}).
		Watches(&source.Channel{Source: EnableUIconfigSync}, &handler.EnqueueRequestForObject{}).
		Complete(r); err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}
	go func() {
		initialSync <- event.GenericEvent{Object: GetPluginDeployment()}
	}()
	return nil
}

// +kubebuilder:rbac:groups="",resources=configmaps;services,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=console.openshift.io,resources=consoleplugins,verbs=get;list;watch;create;update;delete

func (r *ConsolePluginReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	if EnableUI {
		pluginCM := getPluginCM()
		cm := &v1.ConfigMap{
			ObjectMeta: pluginCM.ObjectMeta,
		}
		_, err := applicationset.CreateOrUpdate(ctx, r.Client, cm, func() error {
			if !reflect.DeepEqual(cm.Data, pluginCM.Data) {
				cm.Data = pluginCM.Data
			}
			return nil
		})
		if err != nil {
			return reconcile.Result{}, err
		}

		pluginService := getPluginService()
		service := &v1.Service{
			ObjectMeta: pluginService.ObjectMeta,
		}
		_, err = applicationset.CreateOrUpdate(ctx, r.Client, service, func() error {
			if !reflect.DeepEqual(service.Spec, pluginService.Spec) {
				service.Spec = pluginService.Spec
			}
			return nil
		})
		if err != nil {
			return reconcile.Result{}, err
		}

		pluginDeployment := GetPluginDeployment()
		deployment := &appsv1.Deployment{
			ObjectMeta: pluginDeployment.ObjectMeta,
		}
		_, err = applicationset.CreateOrUpdate(ctx, r.Client, deployment, func() error {
			if !reflect.DeepEqual(deployment.Spec, pluginDeployment.Spec) {
				deployment.Spec = pluginDeployment.Spec
			}
			return nil
		})
		if err != nil {
			return reconcile.Result{}, err
		}

		pluginConsole := getConsolePlugin()
		console := &console.ConsolePlugin{
			ObjectMeta: pluginConsole.ObjectMeta,
		}
		_, err = applicationset.CreateOrUpdate(ctx, r.Client, console, func() error {
			if !reflect.DeepEqual(deployment.Spec, pluginConsole.Spec) {
				console.Spec = pluginConsole.Spec
			}
			return nil
		})
		if err != nil {
			return reconcile.Result{}, err
		}
		err = r.createOrUpdateQuickStarts(ctx, req)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else {
		quickStartClientObjects := getQuickStartClientObjects()
		objects := []client.Object{
			getConsolePlugin(),
			GetPluginDeployment(),
			getPluginService(),
			getPluginCM(),
		}
		objects = append(objects, quickStartClientObjects...)

		for _, obj := range objects {
			if err := r.Client.Delete(ctx, obj); err != nil {
				if !apierrors.IsNotFound(err) {
					return reconcile.Result{}, err
				}
			}
		}
	}

	return reconcile.Result{}, nil
}

func (r *ConsolePluginReconciler) selectPluginDeployment(obj client.Object) bool {
	return obj.GetName() == pluginResourceName && obj.GetNamespace() == pluginNamespace
}

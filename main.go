/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	argo "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	hivev1 "github.com/openshift/hive/apis/hive/v1"
	hypershiftv1alpha1 "github.com/openshift/hypershift/api/v1alpha1"
	v1alpha1 "github.com/stolostron/cluster-templates-operator/api/v1alpha1"
	"github.com/stolostron/cluster-templates-operator/controllers"
	"github.com/stolostron/cluster-templates-operator/controllers/defaultresources"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(hypershiftv1alpha1.AddToScheme(scheme))
	utilruntime.Must(argo.AddToScheme(scheme))
	utilruntime.Must(hivev1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(
		&metricsAddr,
		"metrics-bind-address",
		":8080",
		"The address the metric endpoint binds to.",
	)
	flag.StringVar(
		&probeAddr,
		"health-probe-bind-address",
		":8081",
		"The address the probe endpoint binds to.",
	)
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	config := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "135184d5.openshift.io",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	_, err = mgr.GetClient().RESTMapper().KindFor(schema.GroupVersionResource{
		Group:    "hypershift.openshift.io",
		Resource: "HostedCluster",
	})

	enableHypershift := true
	if err != nil {
		setupLog.Info("Hypershift APIs not found")
		enableHypershift = false
	}

	if err = (&controllers.ClusterTemplateQuotaReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterTemplateQuota")
		os.Exit(1)
	}
	if err = (&controllers.ClusterTemplateInstanceReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		EnableHypershift: enableHypershift,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterTemplateInstance")
		os.Exit(1)
	}

	if enableHypershift {
		if err = (&defaultresources.HypershiftTemplateReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "HypershiftTemplate")
			os.Exit(1)
		}
	}
	/*
		if err = (&defaultresources.HelmRepoReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "HelmRepo")
			os.Exit(1)
		}
	*/

	if err = (&v1alpha1.ClusterTemplateQuota{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "ClusterTemplateQuota")
		os.Exit(1)
	}
	if err = (&v1alpha1.ClusterTemplateInstance{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "ClusterTemplateInstance")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

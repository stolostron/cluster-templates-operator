package clustersetup

import (
	"context"
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	clustertemplatev1alpha1 "github.com/rawagner/cluster-templates-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetClusterSetupDeployment(ctx context.Context, k8sClient client.Client, setupType string) (batchv1.JobSpec, error) {
	if strings.HasPrefix(setupType, "builtin-") {
		jobSpec := batchv1.JobSpec{

			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: "OnFailure",
					Containers: []corev1.Container{
						{
							Name:  "setup",
							Image: "quay.io/rawagner/cluster-setup:latest",
							Env: []corev1.EnvVar{
								{
									Name:  "SETUP_TYPE",
									Value: setupType,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kubeconfig",
									MountPath: "/etc/kubeconfig",
									ReadOnly:  true,
								},
							},
						},
					},
				},
			},
		}
		return jobSpec, nil
	}

	clusterTemplateSetup := clustertemplatev1alpha1.ClusterTemplateSetup{}

	err := k8sClient.Get(ctx, client.ObjectKey{Name: setupType}, &clusterTemplateSetup)

	if err != nil {
		return batchv1.JobSpec{}, err
	}
	return clusterTemplateSetup.Spec.JobSpec, nil
}

func CreateSetupJobs(
	ctx context.Context,
	k8sClient client.Client,
	clusterTemplate clustertemplatev1alpha1.ClusterTemplate,
	clusterTemplateInstance *clustertemplatev1alpha1.ClusterTemplateInstance,
	kubeconfigSecret string,
) error {
	for _, clusterSetup := range clusterTemplate.Spec.ClusterSetup {
		fmt.Println("setup")
		jobSpec, err := GetClusterSetupDeployment(ctx, k8sClient, clusterSetup.Type)
		if err != nil {
			fmt.Println("job spec err", err)
			return err
		}

		jobSpec.Template.Spec.Volumes = append(jobSpec.Template.Spec.Volumes, corev1.Volume{
			Name: "kubeconfig",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: kubeconfigSecret,
				},
			},
		})

		fmt.Println("setup-create")
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: clusterTemplateInstance.Name + "-",
				Namespace:    clusterTemplateInstance.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind:       "ClusterTemplateInstance",
						APIVersion: clustertemplatev1alpha1.APIVersion,
						Name:       clusterTemplateInstance.Name,
						UID:        clusterTemplateInstance.UID,
					},
				},
				Labels: map[string]string{"clusterinstance": clusterTemplateInstance.Name, "setupname": clusterSetup.Name},
			},
			Spec: jobSpec,
		}
		err = k8sClient.Create(ctx, job)
		if err != nil {
			fmt.Println("job create err", err)
			return err
		}
	}
	return nil
}

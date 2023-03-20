package v1alpha1

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ = Describe("ClusterTemplateInstance validating webhook", func() {
	It("Fails ctq list", func() {
		instanceControllerClient = fake.NewFakeClientWithScheme(&runtime.Scheme{})
		cti := ClusterTemplateInstance{
			ObjectMeta: v1.ObjectMeta{
				Namespace: "foo",
			},
		}
		err := cti.ValidateCreate()
		Expect(err).Should(HaveOccurred())
		Expect(
			strings.Contains(err.Error(), "could not list cluster template quotas"),
		).Should(BeTrue())
	})
	It("Fails when ctq does not exit", func() {
		scheme := runtime.NewScheme()
		err := AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())
		instanceControllerClient = fake.NewFakeClientWithScheme(scheme)
		cti := ClusterTemplateInstance{
			ObjectMeta: v1.ObjectMeta{
				Namespace: "foo",
			},
		}
		err = cti.ValidateCreate()
		Expect(err).Should(HaveOccurred())
		Expect(
			err.Error(),
		).Should(Equal("failed quota: no cluster template quota specified for the 'foo' namespace"))
	})
	It("Fails when template does not exists", func() {
		scheme := runtime.NewScheme()
		err := AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())
		ctq := &ClusterTemplateQuota{
			ObjectMeta: v1.ObjectMeta{
				Name:      "bar",
				Namespace: "foo",
			},
			Spec: ClusterTemplateQuotaSpec{
				AllowedTemplates: []AllowedTemplate{
					{
						Name: "foo-tmp",
					},
				},
			},
		}
		instanceControllerClient = fake.NewFakeClientWithScheme(scheme, ctq)
		cti := ClusterTemplateInstance{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo-instance",
				Namespace: "foo",
			},
			Spec: ClusterTemplateInstanceSpec{
				ClusterTemplateRef: "foo-tmp",
			},
		}
		err = cti.ValidateCreate()
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).Should(Equal("cluster template does not exist"))
	})
	It("Fails when quota does not allow template", func() {
		scheme := runtime.NewScheme()
		err := AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())
		ctq := &ClusterTemplateQuota{
			ObjectMeta: v1.ObjectMeta{
				Name:      "bar",
				Namespace: "foo",
			},
			Spec: ClusterTemplateQuotaSpec{
				AllowedTemplates: []AllowedTemplate{
					{
						Name: "bar-tmp",
					},
				},
			},
		}
		ct := &ClusterTemplate{
			ObjectMeta: v1.ObjectMeta{
				Name: "foo-tmp",
			},
		}
		instanceControllerClient = fake.NewFakeClientWithScheme(scheme, ctq, ct)
		cti := ClusterTemplateInstance{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo-instance",
				Namespace: "foo",
			},
			Spec: ClusterTemplateInstanceSpec{
				ClusterTemplateRef: "foo-tmp",
			},
		}
		err = cti.ValidateCreate()
		Expect(err).Should(HaveOccurred())
		Expect(
			err.Error(),
		).Should(Equal("failed quota: quota does not allow 'foo-tmp' cluster template"))
	})
	It("Fails when max insances reached", func() {
		scheme := runtime.NewScheme()
		err := AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())
		ctq := &ClusterTemplateQuota{
			ObjectMeta: v1.ObjectMeta{
				Name:      "bar",
				Namespace: "foo",
			},
			Spec: ClusterTemplateQuotaSpec{
				AllowedTemplates: []AllowedTemplate{
					{
						Name:  "foo-tmp",
						Count: 1,
					},
				},
			},
			Status: ClusterTemplateQuotaStatus{
				TemplateInstances: []AllowedTemplate{
					{
						Name:  "foo-tmp",
						Count: 1,
					},
				},
			},
		}
		ct := &ClusterTemplate{
			ObjectMeta: v1.ObjectMeta{
				Name: "foo-tmp",
			},
		}
		instanceControllerClient = fake.NewFakeClientWithScheme(scheme, ctq, ct)
		cti := ClusterTemplateInstance{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo-instance",
				Namespace: "foo",
			},
			Spec: ClusterTemplateInstanceSpec{
				ClusterTemplateRef: "foo-tmp",
			},
		}
		err = cti.ValidateCreate()
		Expect(err).Should(HaveOccurred())
		Expect(
			err.Error(),
		).Should(Equal("failed quota: cluster instance not allowed - maximum cluster instances reached"))
	})
	It("Passes when ctq allows template", func() {
		scheme := runtime.NewScheme()
		err := AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())
		ctq := &ClusterTemplateQuota{
			ObjectMeta: v1.ObjectMeta{
				Name:      "bar",
				Namespace: "foo",
			},
			Spec: ClusterTemplateQuotaSpec{
				AllowedTemplates: []AllowedTemplate{
					{
						Name: "foo-tmp",
					},
				},
			},
		}
		ct := &ClusterTemplate{
			ObjectMeta: v1.ObjectMeta{
				Name: "foo-tmp",
			},
		}
		instanceControllerClient = fake.NewFakeClientWithScheme(scheme, ctq, ct)
		cti := ClusterTemplateInstance{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo-instance",
				Namespace: "foo",
			},
			Spec: ClusterTemplateInstanceSpec{
				ClusterTemplateRef: "foo-tmp",
			},
		}
		err = cti.ValidateCreate()
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("Fails when updating requester", func() {
		cti := ClusterTemplateInstance{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo-instance",
				Namespace: "foo",
				Annotations: map[string]string{
					CTIRequesterAnnotation: "foo",
				},
			},
			Spec: ClusterTemplateInstanceSpec{
				ClusterTemplateRef: "foo-tmp",
			},
		}

		newCti := &ClusterTemplateInstance{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo-instance",
				Namespace: "foo",
				Annotations: map[string]string{
					CTIRequesterAnnotation: "bar",
				},
			},
			Spec: ClusterTemplateInstanceSpec{
				ClusterTemplateRef: "foo-tmp",
			},
		}

		err := cti.ValidateUpdate(newCti)
		Expect(err).Should(HaveOccurred())
		Expect(
			err.Error(),
		).Should(Equal("cluster requester cannot be changed"))
	})

	It("Fails when updating requester", func() {
		cti := ClusterTemplateInstance{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo-instance",
				Namespace: "foo",
				Annotations: map[string]string{
					CTIRequesterAnnotation: "foo",
				},
			},
			Spec: ClusterTemplateInstanceSpec{
				ClusterTemplateRef: "foo-tmp",
			},
		}

		newCti := &ClusterTemplateInstance{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo-instance",
				Namespace: "foo",
				Annotations: map[string]string{
					CTIRequesterAnnotation: "foo",
				},
			},
			Spec: ClusterTemplateInstanceSpec{
				ClusterTemplateRef: "foo-bar",
			},
		}

		err := cti.ValidateUpdate(newCti)
		Expect(err).Should(HaveOccurred())
		Expect(
			err.Error(),
		).Should(Equal("spec is immutable"))
	})
	It("Succeeds when updating annotations", func() {
		cti := ClusterTemplateInstance{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo-instance",
				Namespace: "foo",
				Annotations: map[string]string{
					CTIRequesterAnnotation: "foo",
				},
			},
			Spec: ClusterTemplateInstanceSpec{
				ClusterTemplateRef: "foo-tmp",
			},
		}

		newCti := &ClusterTemplateInstance{
			ObjectMeta: v1.ObjectMeta{
				Name:      "foo-instance",
				Namespace: "foo",
				Annotations: map[string]string{
					CTIRequesterAnnotation: "foo",
					"foo":                  "bar",
				},
			},
			Spec: ClusterTemplateInstanceSpec{
				ClusterTemplateRef: "foo-tmp",
			},
		}

		err := cti.ValidateUpdate(newCti)
		Expect(err).ShouldNot(HaveOccurred())
	})
})

var _ = Describe("ClusterTemplateInstance mutating webhook", func() {
	It("Fails if template does not exist", func() {
		scheme := runtime.NewScheme()
		err := AddToScheme(scheme)
		Expect(err).ShouldNot(HaveOccurred())
		instanceControllerClient = fake.NewFakeClientWithScheme(scheme)
		cti := &ClusterTemplateInstance{
			ObjectMeta: v1.ObjectMeta{
				Namespace: "foo",
			},
			Spec: ClusterTemplateInstanceSpec{
				ClusterTemplateRef: "foo-template",
			},
		}
		ctx := context.TODO()
		webhookCtx := admission.NewContextWithRequest(ctx, admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				UserInfo: authenticationv1.UserInfo{
					Username: "foo",
				},
			},
		})
		err = cti.Default(webhookCtx, cti)
		Expect(err).Should(HaveOccurred())
	})
	It("Adds finalizer", func() {
		scheme := runtime.NewScheme()
		err := AddToScheme(scheme)
		Expect(err).ShouldNot(HaveOccurred())
		ct := &ClusterTemplate{
			ObjectMeta: v1.ObjectMeta{
				Name: "foo-template",
			},
		}
		instanceControllerClient = fake.NewFakeClientWithScheme(scheme, ct)
		cti := &ClusterTemplateInstance{
			ObjectMeta: v1.ObjectMeta{
				Namespace: "foo",
			},
			Spec: ClusterTemplateInstanceSpec{
				ClusterTemplateRef: "foo-template",
			},
		}
		ctx := context.TODO()
		webhookCtx := admission.NewContextWithRequest(ctx, admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				UserInfo: authenticationv1.UserInfo{
					Username: "foo",
				},
			},
		})
		err = cti.Default(webhookCtx, cti)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(controllerutil.ContainsFinalizer(cti, CTIFinalizer)).Should(BeTrue())
		Expect(cti.Annotations[CTIRequesterAnnotation]).Should(Equal("foo"))
	})

	It("Adds experimetal provider annotation", func() {
		scheme := runtime.NewScheme()
		err := AddToScheme(scheme)
		Expect(err).ShouldNot(HaveOccurred())
		ct := &ClusterTemplate{
			ObjectMeta: v1.ObjectMeta{
				Name: "foo-template",
				Annotations: map[string]string{
					ClusterProviderExperimentalAnnotation: "true",
				},
			},
		}
		instanceControllerClient = fake.NewFakeClientWithScheme(scheme, ct)
		cti := &ClusterTemplateInstance{
			ObjectMeta: v1.ObjectMeta{
				Namespace: "foo",
			},
			Spec: ClusterTemplateInstanceSpec{
				ClusterTemplateRef: "foo-template",
			},
		}
		ctx := context.TODO()
		webhookCtx := admission.NewContextWithRequest(ctx, admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				UserInfo: authenticationv1.UserInfo{
					Username: "foo",
				},
			},
		})
		err = cti.Default(webhookCtx, cti)
		Expect(err).ShouldNot(HaveOccurred())
		value, ok := cti.Annotations[ClusterProviderExperimentalAnnotation]
		Expect(ok).To(BeTrue())
		Expect(value).To(Equal("true"))
	})
})

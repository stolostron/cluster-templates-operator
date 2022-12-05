package v1alpha1

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("ClusterTemplateInstance webhook", func() {
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
})

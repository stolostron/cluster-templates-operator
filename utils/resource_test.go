package utils

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Resource utils", func() {
	scheme := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(scheme))
	It("EnsureResourceExists", func() {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
			Data: map[string][]byte{
				"key": []byte("value"),
			},
		}
		client := fake.NewFakeClientWithScheme(scheme)
		err := EnsureResourceExists(context.TODO(), client, secret, false)
		Expect(err).To(BeNil())

		secretMeta := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "bar",
			},
		}

		err = EnsureResourceExists(context.TODO(), client, secretMeta, true)
		Expect(err).To(BeNil())
		Expect(secretMeta.Data["key"]).To(Equal([]byte("value")))
	})
})

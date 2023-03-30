package utils

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func EnsureResourceExists(
	ctx context.Context,
	k8sClient client.Client,
	obj client.Object,
	loadBack bool,
) error {
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		if err = k8sClient.Create(ctx, obj); err != nil {
			return err
		}
	}
	if loadBack {
		return k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj)
	}
	return nil
}

func DeleteIfExists(
	ctx context.Context,
	k8sClient client.Client,
	obj client.Object,
) error {
	if obj == nil {
		return nil
	}
	if err := k8sClient.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

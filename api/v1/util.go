package v1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	resourcehelper "k8s.io/component-helpers/resource"
)

// PodEffectiveRequests returns the pod's effective resource requests, accounting
// for init and sidecar containers the same way the native Kubernetes
// ResourceQuota admission does: for each resource the effective amount is the
// larger of the sum across regular containers and the amount held by init
// containers (always-restarting sidecar containers are added to the sum).
//
// Pod overhead (pod.Spec.Overhead, populated from the RuntimeClass for
// sandboxed runtimes such as Kata or gVisor) is included, matching the native
// quota. For the common runc runtime overhead is empty and contributes nothing.
func PodEffectiveRequests(pod *corev1.Pod) corev1.ResourceList {
	return resourcehelper.PodRequests(pod, resourcehelper.PodResourcesOptions{})
}

// PodEffectiveLimits returns the pod's effective resource limits, following the
// same init/sidecar-aware rules (and pod overhead inclusion) as
// PodEffectiveRequests.
func PodEffectiveLimits(pod *corev1.Pod) corev1.ResourceList {
	return resourcehelper.PodLimits(pod, resourcehelper.PodResourcesOptions{})
}

func AddFinalizer(name string, obj runtime.Object) error {
	metadata, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	exists := false
	finalizers := metadata.GetFinalizers()
	for _, f := range finalizers {
		if f == name {
			exists = true
			break
		}
	}
	if !exists {
		metadata.SetFinalizers(append(finalizers, name))
	}

	return nil
}

func RemoveFinalizer(name string, obj runtime.Object) error {
	metadata, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	var finalizers []string
	for _, finalizer := range metadata.GetFinalizers() {
		if finalizer == name {
			continue
		}
		finalizers = append(finalizers, finalizer)
	}
	metadata.SetFinalizers(finalizers)

	return nil
}

func IsAnnotationExists(obj runtime.Object, key string) bool {
	metadata, err := meta.Accessor(obj)
	if err != nil {
		return false
	}

	_, found := metadata.GetAnnotations()[key]
	return found
}

func AddAnnotation(obj runtime.Object, key, val string) error {
	objMeta, err := meta.Accessor(obj)
	if err != nil {
		return fmt.Errorf("cannot add annotation for invalid object %v: %v", obj, err)
	}

	annos := objMeta.GetAnnotations()
	if annos == nil {
		annos = map[string]string{}
	}
	annos[key] = val
	objMeta.SetAnnotations(annos)

	return nil
}

func RemoveAnnotation(obj runtime.Object, key string) error {
	metadata, err := meta.Accessor(obj)
	if err != nil {
		return fmt.Errorf("cannot remove annotation for invalid object %v: %v", obj, err)
	}

	annos := metadata.GetAnnotations()
	delete(annos, key)
	metadata.SetAnnotations(annos)

	return nil
}

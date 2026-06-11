package v1

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func SetupPodWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &corev1.Pod{}).
		WithDefaulter(&podAnnotator{mgr.GetClient()}).
		WithValidator(&podValidator{mgr.GetClient()}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=ignore,sideEffects=None,groups="",matchPolicy=Exact,resources=pods,verbs=create;update,versions=v1,name=pod.jenting.io,admissionReviewVersions=v1

// podAnnotator annotates Pods
type podAnnotator struct {
	client.Client
}

func (a *podAnnotator) Default(ctx context.Context, pod *corev1.Pod) error {
	log := logf.FromContext(ctx)

	// check whether one of the projectresourcequotas.jenting.io CR spec.hard.pods is set
	prqList := &ProjectResourceQuotaList{}
	if err := a.Client.List(ctx, prqList); err != nil {
		return err
	}

	for _, prq := range prqList.Items {
		// skip the projectresourcequota CR that is being deleted
		if prq.DeletionTimestamp != nil {
			continue
		}

		for _, ns := range prq.Spec.Namespaces {
			if ns != pod.Namespace {
				// skip the namespace that is not within the prq.spec.namespaces
				continue
			}

			_, ok := prq.Spec.Hard[corev1.ResourcePods]
			if !ok {
				return nil
			}

			if err := AddAnnotation(pod, ProjectResourceQuotaAnnotation, prq.Name); err != nil {
				return err
			}
			log.Info("Pod annotated")
			return nil
		}
	}

	return nil
}

//+kubebuilder:webhook:path=/validate--v1-pod,mutating=false,failurePolicy=ignore,sideEffects=None,groups="",matchPolicy=Exact,resources=pods,verbs=create;update,versions=v1,name=pod.jenting.io,admissionReviewVersions=v1

// podValidator validates Pods
type podValidator struct {
	client.Client
}

func (v *podValidator) ValidateCreate(ctx context.Context, pod *corev1.Pod) (admission.Warnings, error) {
	log := logf.FromContext(ctx)

	log.Info("Validating Pod creation")
	prqName, found := pod.Annotations[ProjectResourceQuotaAnnotation]
	if !found {
		return nil, nil
	}

	// get the current projectresourcequotas.jenting.io CR
	prq := &ProjectResourceQuota{}
	if err := v.Client.Get(ctx, types.NamespacedName{Name: prqName}, prq); err != nil {
		return nil, err
	}

	// check the status.used.pods is less than spec.hard.pods
	used := prq.Status.Used[corev1.ResourcePods]
	hard := prq.Spec.Hard[corev1.ResourcePods]
	if hard.Cmp(used) != 1 {
		return nil, fmt.Errorf("over project resource quota. current %s counts %v, hard limit count %v", corev1.ResourcePods, used.String(), hard.String())
	}

	// calculate the pod's effective resource requests and limits, accounting
	// for init and sidecar containers like the native ResourceQuota does
	requests := PodEffectiveRequests(pod)
	limits := PodEffectiveLimits(pod)

	// checkQuota rejects the pod when its request for the resource, added to the
	// project's current usage, would exceed the hard limit. The reported request
	// is the incoming pod's amount (before adding the existing usage).
	checkQuota := func(name corev1.ResourceName, request resource.Quantity) error {
		used := prq.Status.Used[name]
		hard := prq.Spec.Hard[name]
		total := request.DeepCopy()
		total.Add(used)
		if total.Cmp(hard) == 1 {
			return fmt.Errorf("over project resource quota. %s request %s + used %s > hard limit %s", name, request.String(), used.String(), hard.String())
		}
		return nil
	}

	// "cpu"/"memory"/... without the requests prefix map to the same request values
	for _, check := range []struct {
		name    corev1.ResourceName
		request resource.Quantity
	}{
		{corev1.ResourceCPU, requests[corev1.ResourceCPU]},
		{corev1.ResourceMemory, requests[corev1.ResourceMemory]},
		{corev1.ResourceStorage, requests[corev1.ResourceStorage]},
		{corev1.ResourceEphemeralStorage, requests[corev1.ResourceEphemeralStorage]},
		{corev1.ResourceRequestsCPU, requests[corev1.ResourceCPU]},
		{corev1.ResourceRequestsMemory, requests[corev1.ResourceMemory]},
		{corev1.ResourceRequestsStorage, requests[corev1.ResourceStorage]},
		{corev1.ResourceRequestsEphemeralStorage, requests[corev1.ResourceEphemeralStorage]},
		{corev1.ResourceLimitsCPU, limits[corev1.ResourceCPU]},
		{corev1.ResourceLimitsMemory, limits[corev1.ResourceMemory]},
		{corev1.ResourceLimitsEphemeralStorage, limits[corev1.ResourceEphemeralStorage]},
	} {
		if err := checkQuota(check.name, check.request); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (v *podValidator) ValidateUpdate(ctx context.Context, oldObj, newObj *corev1.Pod) (admission.Warnings, error) {
	// we don't need to validate resources.requests.* and resources.limits.* update
	// because Kubernetes does not allow to update them
	return nil, nil
}

func (v *podValidator) ValidateDelete(ctx context.Context, pod *corev1.Pod) (admission.Warnings, error) {
	return nil, nil
}

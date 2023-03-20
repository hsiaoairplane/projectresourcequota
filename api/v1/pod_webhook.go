/*
Copyright 2023 JenTing.

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

package v1

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func SetupPodWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&corev1.Pod{}).
		WithDefaulter(&podAnnotator{mgr.GetClient()}).
		WithValidator(&podValidator{mgr.GetClient()}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=ignore,sideEffects=None,groups="",matchPolicy=Exact,resources=pods,verbs=create;update,versions=v1,name=pod.jenting.io,admissionReviewVersions=v1

// podAnnotator annotates Pods
type podAnnotator struct {
	client.Client
}

func (a *podAnnotator) Default(ctx context.Context, obj runtime.Object) error {
	log := logf.FromContext(ctx)
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected a Pod but got a %T", obj)
	}

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

			AddAnnotation(pod, ProjectResourceQuotaAnnotation, prq.Name)
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

func (v *podValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	log := logf.FromContext(ctx)
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected a Pod but got a %T", obj)
	}

	log.Info("Validating Pod creation")
	prqName, found := pod.Annotations[ProjectResourceQuotaAnnotation]
	if !found {
		return nil
	}

	// get the current projectresourcequotas.jenting.io CR
	prq := &ProjectResourceQuota{}
	if err := v.Client.Get(ctx, types.NamespacedName{Name: prqName}, prq); err != nil {
		return err
	}

	// check the status.used.pods is less than spec.hard.pods
	hard := prq.Spec.Hard[corev1.ResourcePods]
	if hard.Cmp(prq.Status.Used[corev1.ResourcePods]) != 1 {
		return fmt.Errorf("over project resource quota. current %s counts %v, hard limit count %v", corev1.ResourcePods, hard, prq.Status.Used[corev1.ResourcePods])
	}

	// calculate resource requests and limits
	var cpu, memory, storage, ephemeralStorage resource.Quantity
	var requestCPU, requestMemory, requestStorage, requestEphemeralStorage resource.Quantity
	var limitCPU, limitMemory, limitEphemeralStorage resource.Quantity
	for _, container := range pod.Spec.Containers {
		// without requests prefix
		cpu.Add(container.Resources.Requests[corev1.ResourceCPU])
		memory.Add(container.Resources.Requests[corev1.ResourceMemory])
		storage.Add(container.Resources.Requests[corev1.ResourceStorage])
		ephemeralStorage.Add(container.Resources.Requests[corev1.ResourceEphemeralStorage])

		// with requests prefix
		requestCPU.Add(container.Resources.Requests[corev1.ResourceRequestsCPU])
		requestMemory.Add(container.Resources.Requests[corev1.ResourceRequestsMemory])
		requestStorage.Add(container.Resources.Requests[corev1.ResourceRequestsStorage])
		requestEphemeralStorage.Add(container.Resources.Requests[corev1.ResourceRequestsEphemeralStorage])

		// limits
		limitCPU.Add(container.Resources.Limits[corev1.ResourceLimitsCPU])
		limitMemory.Add(container.Resources.Limits[corev1.ResourceLimitsMemory])
		limitEphemeralStorage.Add(container.Resources.Limits[corev1.ResourceLimitsEphemeralStorage])
	}

	// without requests prefix
	cpu.Add(prq.Status.Used[corev1.ResourceCPU])
	if cpu.Cmp(prq.Spec.Hard[corev1.ResourceCPU]) == 1 {
		return fmt.Errorf("over project resource quota. %s request %v + used %v > hard limit %v", corev1.ResourceCPU, cpu, prq.Status.Used[corev1.ResourceCPU], prq.Spec.Hard[corev1.ResourceCPU])
	}
	memory.Add(prq.Status.Used[corev1.ResourceMemory])
	if memory.Cmp(prq.Spec.Hard[corev1.ResourceMemory]) == 1 {
		return fmt.Errorf("over project resource quota. %s request %v + used %v > hard limit %v", corev1.ResourceMemory, memory, prq.Status.Used[corev1.ResourceMemory], prq.Spec.Hard[corev1.ResourceMemory])
	}
	storage.Add(prq.Status.Used[corev1.ResourceStorage])
	if storage.Cmp(prq.Spec.Hard[corev1.ResourceStorage]) == 1 {
		return fmt.Errorf("over project resource quota. %s request %v + used %v > hard limit %v", corev1.ResourceStorage, storage, prq.Status.Used[corev1.ResourceStorage], prq.Spec.Hard[corev1.ResourceStorage])
	}
	ephemeralStorage.Add(prq.Status.Used[corev1.ResourceEphemeralStorage])
	if ephemeralStorage.Cmp(prq.Spec.Hard[corev1.ResourceEphemeralStorage]) == 1 {
		return fmt.Errorf("over project resource quota. %s request %v + used %v > hard limit %v", corev1.ResourceEphemeralStorage, ephemeralStorage, prq.Status.Used[corev1.ResourceEphemeralStorage], prq.Spec.Hard[corev1.ResourceEphemeralStorage])
	}

	// with requests prefix
	requestCPU.Add(prq.Status.Used[corev1.ResourceRequestsCPU])
	if requestCPU.Cmp(prq.Spec.Hard[corev1.ResourceRequestsCPU]) == 1 {
		return fmt.Errorf("over project resource quota. %s request %v + used %v > hard limit %v", corev1.ResourceRequestsCPU, requestCPU, prq.Status.Used[corev1.ResourceRequestsCPU], prq.Spec.Hard[corev1.ResourceRequestsCPU])
	}
	requestMemory.Add(prq.Status.Used[corev1.ResourceRequestsMemory])
	if requestMemory.Cmp(prq.Spec.Hard[corev1.ResourceRequestsMemory]) == 1 {
		return fmt.Errorf("over project resource quota. %s request %v + used %v > hard limit %v", corev1.ResourceRequestsMemory, requestMemory, prq.Status.Used[corev1.ResourceRequestsMemory], prq.Spec.Hard[corev1.ResourceRequestsMemory])
	}
	requestStorage.Add(prq.Status.Used[corev1.ResourceRequestsStorage])
	if requestStorage.Cmp(prq.Spec.Hard[corev1.ResourceRequestsStorage]) == 1 {
		return fmt.Errorf("over project resource quota. %s request %v + used %v > hard limit %v", corev1.ResourceRequestsStorage, requestStorage, prq.Status.Used[corev1.ResourceRequestsStorage], prq.Spec.Hard[corev1.ResourceRequestsStorage])
	}
	requestEphemeralStorage.Add(prq.Status.Used[corev1.ResourceEphemeralStorage])
	if requestEphemeralStorage.Cmp(prq.Spec.Hard[corev1.ResourceEphemeralStorage]) == 1 {
		return fmt.Errorf("over project resource quota. %s request %v + used %v > hard limit %v", corev1.ResourceRequestsEphemeralStorage, requestEphemeralStorage, prq.Status.Used[corev1.ResourceEphemeralStorage], prq.Spec.Hard[corev1.ResourceEphemeralStorage])
	}

	// limits
	limitCPU.Add(prq.Status.Used[corev1.ResourceLimitsCPU])
	if limitCPU.Cmp(prq.Spec.Hard[corev1.ResourceLimitsCPU]) == 1 {
		return fmt.Errorf("over project resource quota. %s request %v + used %v > hard limit %v", corev1.ResourceLimitsCPU, requestCPU, prq.Status.Used[corev1.ResourceLimitsCPU], prq.Spec.Hard[corev1.ResourceLimitsCPU])
	}
	limitMemory.Add(prq.Status.Used[corev1.ResourceLimitsMemory])
	if limitMemory.Cmp(prq.Spec.Hard[corev1.ResourceLimitsMemory]) == 1 {
		return fmt.Errorf("over project resource quota. %s request %v + used %v > hard limit %v", corev1.ResourceLimitsMemory, requestMemory, prq.Status.Used[corev1.ResourceLimitsMemory], prq.Spec.Hard[corev1.ResourceLimitsMemory])
	}
	limitEphemeralStorage.Add(prq.Status.Used[corev1.ResourceLimitsEphemeralStorage])
	if limitEphemeralStorage.Cmp(prq.Spec.Hard[corev1.ResourceLimitsEphemeralStorage]) == 1 {
		return fmt.Errorf("over project resource quota. %s request %v + used %v > hard limit %v", corev1.ResourceRequestsEphemeralStorage, requestEphemeralStorage, prq.Status.Used[corev1.ResourceLimitsEphemeralStorage], prq.Spec.Hard[corev1.ResourceLimitsEphemeralStorage])
	}
	return nil
}

func (v *podValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	// we don't need to validate resources.requests.* and resources.limits.* update
	// because Kubernetes does not allow to update them
	return nil
}

func (v *podValidator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

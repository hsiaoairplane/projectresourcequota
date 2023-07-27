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
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
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

func (v *podValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	log := logf.FromContext(ctx)
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("expected a Pod but got a %T", obj)
	}

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
		return nil, fmt.Errorf("over project resource quota. current %s counts %v, hard limit count %v", corev1.ResourcePods, hard.String(), used.String())
	}

	// calculate resource requests and limits
	var cpu, memory, storage, ephemeralStorage resource.Quantity
	var requestCPU, requestMemory, requestStorage, requestEphemeralStorage resource.Quantity
	var limitCPU, limitMemory, limitEphemeralStorage resource.Quantity
	for _, container := range pod.Spec.Containers {
		// without requests prefix
		quality := container.Resources.Requests.Cpu()
		if quality != nil {
			cpu.Add(*quality)
			requestCPU.Add(*quality)
		}
		quality = container.Resources.Requests.Memory()
		if quality != nil {
			memory.Add(*quality)
			requestMemory.Add(*quality)
		}
		quality = container.Resources.Requests.Storage()
		if quality != nil {
			storage.Add(*quality)
			requestStorage.Add(*quality)
		}
		quality = container.Resources.Requests.StorageEphemeral()
		if quality != nil {
			ephemeralStorage.Add(*quality)
			requestEphemeralStorage.Add(*quality)
		}

		// limits
		quality = container.Resources.Limits.Cpu()
		if quality != nil {
			limitCPU.Add(*quality)
		}
		quality = container.Resources.Limits.Memory()
		if quality != nil {
			limitMemory.Add(*quality)
		}
		quality = container.Resources.Limits.StorageEphemeral()
		if quality != nil {
			limitEphemeralStorage.Add(*quality)
		}
	}

	// without requests prefix
	used = prq.Status.Used[corev1.ResourceCPU]
	hard = prq.Spec.Hard[corev1.ResourceCPU]
	cpu.Add(used)
	if cpu.Cmp(hard) == 1 {
		return nil, fmt.Errorf("over project resource quota. %s request %s + used %s > hard limit %s", corev1.ResourceCPU, cpu.String(), used.String(), hard.String())
	}

	used = prq.Status.Used[corev1.ResourceMemory]
	hard = prq.Spec.Hard[corev1.ResourceMemory]
	memory.Add(used)
	if memory.Cmp(hard) == 1 {
		return nil, fmt.Errorf("over project resource quota. %s request %s + used %s > hard limit %s", corev1.ResourceMemory, memory.String(), used.String(), hard.String())
	}

	used = prq.Status.Used[corev1.ResourceStorage]
	hard = prq.Spec.Hard[corev1.ResourceStorage]
	storage.Add(used)
	if storage.Cmp(hard) == 1 {
		return nil, fmt.Errorf("over project resource quota. %s request %s + used %s > hard limit %s", corev1.ResourceStorage, storage.String(), used.String(), hard.String())
	}

	used = prq.Status.Used[corev1.ResourceEphemeralStorage]
	hard = prq.Spec.Hard[corev1.ResourceEphemeralStorage]
	ephemeralStorage.Add(used)
	if ephemeralStorage.Cmp(hard) == 1 {
		return nil, fmt.Errorf("over project resource quota. %s request %s + used %s > hard limit %s", corev1.ResourceEphemeralStorage, ephemeralStorage.String(), used.String(), hard.String())
	}

	// with requests prefix
	used = prq.Status.Used[corev1.ResourceRequestsCPU]
	hard = prq.Spec.Hard[corev1.ResourceRequestsCPU]
	requestCPU.Add(used)
	if requestCPU.Cmp(hard) == 1 {
		return nil, fmt.Errorf("over project resource quota. %s request %s + used %s > hard limit %s", corev1.ResourceRequestsCPU, requestCPU.String(), used.String(), hard.String())
	}

	used = prq.Status.Used[corev1.ResourceRequestsMemory]
	hard = prq.Spec.Hard[corev1.ResourceRequestsMemory]
	requestMemory.Add(used)
	if requestMemory.Cmp(hard) == 1 {
		return nil, fmt.Errorf("over project resource quota. %s request %s + used %s > hard limit %s", corev1.ResourceRequestsMemory, requestMemory.String(), used.String(), hard.String())
	}

	used = prq.Status.Used[corev1.ResourceRequestsStorage]
	hard = prq.Spec.Hard[corev1.ResourceRequestsStorage]
	requestStorage.Add(used)
	if requestStorage.Cmp(hard) == 1 {
		return nil, fmt.Errorf("over project resource quota. %s request %s + used %s > hard limit %s", corev1.ResourceRequestsStorage, requestStorage.String(), used.String(), hard.String())
	}

	used = prq.Status.Used[corev1.ResourceRequestsEphemeralStorage]
	hard = prq.Spec.Hard[corev1.ResourceRequestsEphemeralStorage]
	requestEphemeralStorage.Add(used)
	if requestEphemeralStorage.Cmp(hard) == 1 {
		return nil, fmt.Errorf("over project resource quota. %s request %s + used %s > hard limit %s", corev1.ResourceRequestsEphemeralStorage, requestEphemeralStorage.String(), used.String(), hard.String())
	}

	// limits
	used = prq.Status.Used[corev1.ResourceLimitsCPU]
	hard = prq.Spec.Hard[corev1.ResourceLimitsCPU]
	limitCPU.Add(used)
	if limitCPU.Cmp(hard) == 1 {
		return nil, fmt.Errorf("over project resource quota. %s request %v + used %v > hard limit %v", corev1.ResourceLimitsCPU, requestCPU.String(), used.String(), hard.String())
	}

	used = prq.Status.Used[corev1.ResourceLimitsMemory]
	hard = prq.Spec.Hard[corev1.ResourceLimitsMemory]
	limitMemory.Add(used)
	if limitMemory.Cmp(hard) == 1 {
		return nil, fmt.Errorf("over project resource quota. %s request %v + used %v > hard limit %v", corev1.ResourceLimitsMemory, requestMemory.String(), used.String(), hard.String())
	}

	used = prq.Status.Used[corev1.ResourceLimitsEphemeralStorage]
	hard = prq.Spec.Hard[corev1.ResourceLimitsEphemeralStorage]
	limitEphemeralStorage.Add(used)
	if limitEphemeralStorage.Cmp(hard) == 1 {
		return nil, fmt.Errorf("over project resource quota. %s request %v + used %v > hard limit %v", corev1.ResourceRequestsEphemeralStorage, requestEphemeralStorage.String(), used.String(), hard.String())
	}
	return nil, nil
}

func (v *podValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	// we don't need to validate resources.requests.* and resources.limits.* update
	// because Kubernetes does not allow to update them
	return nil, nil
}

func (v *podValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

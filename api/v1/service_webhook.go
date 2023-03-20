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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func SetupServiceWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&corev1.Pod{}).
		WithDefaulter(&serviceAnnotator{mgr.GetClient()}).
		WithValidator(&serviceValidator{mgr.GetClient()}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate--v1-service,mutating=true,failurePolicy=ignore,sideEffects=None,groups="",matchPolicy=Exact,resources=services,verbs=create;update,versions=v1,name=service.jenting.io,admissionReviewVersions=v1

// serviceAnnotator annotates Services
type serviceAnnotator struct {
	client.Client
}

func (a *serviceAnnotator) Default(ctx context.Context, obj runtime.Object) error {
	log := logf.FromContext(ctx)
	svc, ok := obj.(*corev1.Service)
	if !ok {
		return fmt.Errorf("expected a Service but got a %T", obj)
	}

	// check whether one of the projectresourcequotas.jenting.io CR spec.hard.services is set
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
			// skip the namespace that is not within the prq.spec.namespaces
			if ns != svc.Namespace {
				continue
			}

			_, ok := prq.Spec.Hard[corev1.ResourceServices]
			if !ok {
				return nil
			}

			if svc.Annotations == nil {
				svc.Annotations = map[string]string{}
			}
			svc.Annotations[ProjectResourceQuotaLabel] = prq.Name
			log.Info("Service Labeled")
			return nil
		}
	}

	return nil
}

//+kubebuilder:webhook:path=/validate--v1-service,mutating=false,failurePolicy=ignore,sideEffects=None,groups="",matchPolicy=Exact,resources=services,verbs=create;update,versions=v1,name=service.jenting.io,admissionReviewVersions=v1

// serviceValidator validates Services
type serviceValidator struct {
	client.Client
}

func (v *serviceValidator) validateService(ctx context.Context, obj runtime.Object, prq *ProjectResourceQuota) error {
	// check the status.used.services is less than spec.hard.services
	hard := prq.Spec.Hard[corev1.ResourceServices]
	used := prq.Status.Used[corev1.ResourceServices]

	if hard.Cmp(prq.Status.Used[corev1.ResourceServices]) != 1 {
		return fmt.Errorf("over project resource quota. current %s counts %s, hard limit count %s", corev1.ResourceServices, hard.String(), used.String())
	}
	return nil
}

func (v *serviceValidator) validateServiceNodePort(ctx context.Context, obj runtime.Object, prq *ProjectResourceQuota) error {
	// check the status.used.services.nodeports is less than spec.hard.services.nodeports
	hard := prq.Spec.Hard[corev1.ResourceServicesNodePorts]
	used := prq.Status.Used[corev1.ResourceServicesNodePorts]

	if hard.Cmp(prq.Status.Used[corev1.ResourceServicesNodePorts]) != 1 {
		return fmt.Errorf("over project resource quota. current %s counts %s, hard limit count %s", corev1.ResourceServicesNodePorts, hard.String(), used.String())
	}
	return nil
}

func (v *serviceValidator) validateServiceLoadBalancer(ctx context.Context, obj runtime.Object, prq *ProjectResourceQuota) error {
	// check the status.used.services.loadbalancers is less than spec.hard.services.loadbalancers
	hard := prq.Spec.Hard[corev1.ResourceServicesLoadBalancers]
	used := prq.Status.Used[corev1.ResourceServicesLoadBalancers]

	if hard.Cmp(prq.Status.Used[corev1.ResourceServicesLoadBalancers]) != 1 {
		return fmt.Errorf("over project resource quota. current %s counts %s, hard limit count %s", corev1.ResourceServicesLoadBalancers, hard.String(), used.String())
	}
	return nil
}

func (v *serviceValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	log := logf.FromContext(ctx)
	svc, ok := obj.(*corev1.Service)
	if !ok {
		return fmt.Errorf("expected a Service but got a %T", obj)
	}

	log.Info("Validating Service Creates")
	prqName, found := svc.Annotations[ProjectResourceQuotaLabel]
	if !found {
		return nil
	}

	// get the current projectresourcequotas.jenting.io CR
	prq := &ProjectResourceQuota{}
	if err := v.Client.Get(ctx, types.NamespacedName{Name: prqName}, prq); err != nil {
		return err
	}

	// validate services
	if err := v.validateService(ctx, obj, prq); err != nil {
		return err
	}
	switch svc.Spec.Type {
	case corev1.ServiceTypeNodePort:
		// validate services.nodeports
		return v.validateServiceNodePort(ctx, obj, prq)
	case corev1.ServiceTypeLoadBalancer:
		// validate services.loadbalancers
		return v.validateServiceLoadBalancer(ctx, obj, prq)
	}
	return nil
}

func (v *serviceValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	log := logf.FromContext(ctx)
	oldSvc, ok := newObj.(*corev1.Service)
	if !ok {
		return fmt.Errorf("expected a Service but got a %T", oldSvc)
	}
	newSvc, ok := newObj.(*corev1.Service)
	if !ok {
		return fmt.Errorf("expected a Service but got a %T", newObj)
	}

	log.Info("Validating Service Updates")
	prqName, found := newSvc.Annotations[ProjectResourceQuotaLabel]
	if !found {
		return fmt.Errorf("missing annotation %s", ProjectResourceQuotaLabel)
	}

	// get the current projectresourcequotas.jenting.io CR
	prq := &ProjectResourceQuota{}
	if err := v.Client.Get(ctx, types.NamespacedName{Name: prqName}, prq); err != nil {
		return err
	}

	if err := v.validateService(ctx, newObj, prq); err != nil {
		return err
	}

	if newSvc.Spec.Type == corev1.ServiceTypeNodePort && oldSvc.Spec.Type != corev1.ServiceTypeNodePort {
		return v.validateServiceNodePort(ctx, newObj, prq)
	}
	if newSvc.Spec.Type == corev1.ServiceTypeLoadBalancer && oldSvc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return v.validateServiceLoadBalancer(ctx, newObj, prq)
	}
	return nil
}

func (v *serviceValidator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

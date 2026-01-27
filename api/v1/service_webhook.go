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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func SetupServiceWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &corev1.Service{}).
		WithDefaulter(&serviceAnnotator{mgr.GetClient()}).
		WithValidator(&serviceValidator{mgr.GetClient()}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate--v1-service,mutating=true,failurePolicy=ignore,sideEffects=None,groups="",matchPolicy=Exact,resources=services,verbs=create;update,versions=v1,name=service.jenting.io,admissionReviewVersions=v1

// serviceAnnotator annotates Services
type serviceAnnotator struct {
	client.Client
}

func (a *serviceAnnotator) Default(ctx context.Context, svc *corev1.Service) error {
	log := logf.FromContext(ctx)

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

			AddAnnotation(svc, ProjectResourceQuotaAnnotation, prq.Name)
			log.Info("Service annotated")
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

func (v *serviceValidator) validateService(ctx context.Context, svc *corev1.Service, prq *ProjectResourceQuota) error {
	// check the status.used.services is less than spec.hard.services
	hard := prq.Spec.Hard[corev1.ResourceServices]
	used := prq.Status.Used[corev1.ResourceServices]

	if hard.Cmp(prq.Status.Used[corev1.ResourceServices]) != 1 {
		return fmt.Errorf("over project resource quota. current %s counts %s, hard limit count %s", corev1.ResourceServices, hard.String(), used.String())
	}
	return nil
}

func (v *serviceValidator) validateServiceNodePort(ctx context.Context, svc *corev1.Service, prq *ProjectResourceQuota) error {
	// check the status.used.services.nodeports is less than spec.hard.services.nodeports
	hard := prq.Spec.Hard[corev1.ResourceServicesNodePorts]
	used := prq.Status.Used[corev1.ResourceServicesNodePorts]

	if hard.Cmp(prq.Status.Used[corev1.ResourceServicesNodePorts]) != 1 {
		return fmt.Errorf("over project resource quota. current %s counts %s, hard limit count %s", corev1.ResourceServicesNodePorts, hard.String(), used.String())
	}
	return nil
}

func (v *serviceValidator) validateServiceLoadBalancer(ctx context.Context, svc *corev1.Service, prq *ProjectResourceQuota) error {
	// check the status.used.services.loadbalancers is less than spec.hard.services.loadbalancers
	hard := prq.Spec.Hard[corev1.ResourceServicesLoadBalancers]
	used := prq.Status.Used[corev1.ResourceServicesLoadBalancers]

	if hard.Cmp(prq.Status.Used[corev1.ResourceServicesLoadBalancers]) != 1 {
		return fmt.Errorf("over project resource quota. current %s counts %s, hard limit count %s", corev1.ResourceServicesLoadBalancers, hard.String(), used.String())
	}
	return nil
}

func (v *serviceValidator) ValidateCreate(ctx context.Context, svc *corev1.Service) (admission.Warnings, error) {
	log := logf.FromContext(ctx)

	log.Info("Validating Service creation")
	prqName, found := svc.Annotations[ProjectResourceQuotaAnnotation]
	if !found {
		return nil, nil
	}

	// get the current projectresourcequotas.jenting.io CR
	prq := &ProjectResourceQuota{}
	if err := v.Client.Get(ctx, types.NamespacedName{Name: prqName}, prq); err != nil {
		return nil, err
	}

	// validate services
	if err := v.validateService(ctx, svc, prq); err != nil {
		return nil, err
	}
	switch svc.Spec.Type {
	case corev1.ServiceTypeNodePort:
		// validate services.nodeports
		return nil, v.validateServiceNodePort(ctx, svc, prq)
	case corev1.ServiceTypeLoadBalancer:
		// validate services.loadbalancers
		return nil, v.validateServiceLoadBalancer(ctx, svc, prq)
	}
	return nil, nil
}

func (v *serviceValidator) ValidateUpdate(ctx context.Context, oldSvc, newSvc *corev1.Service) (admission.Warnings, error) {
	log := logf.FromContext(ctx)

	log.Info("Validating Service creation")
	prqName, found := newSvc.Annotations[ProjectResourceQuotaAnnotation]
	if !found {
		return nil, fmt.Errorf("missing annotation %s", ProjectResourceQuotaAnnotation)
	}

	// get the current projectresourcequotas.jenting.io CR
	prq := &ProjectResourceQuota{}
	if err := v.Client.Get(ctx, types.NamespacedName{Name: prqName}, prq); err != nil {
		return nil, err
	}

	if err := v.validateService(ctx, newSvc, prq); err != nil {
		return nil, err
	}

	if newSvc.Spec.Type == corev1.ServiceTypeNodePort && oldSvc.Spec.Type != corev1.ServiceTypeNodePort {
		return nil, v.validateServiceNodePort(ctx, newSvc, prq)
	}
	if newSvc.Spec.Type == corev1.ServiceTypeLoadBalancer && oldSvc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return nil, v.validateServiceLoadBalancer(ctx, newSvc, prq)
	}
	return nil, nil
}

func (v *serviceValidator) ValidateDelete(ctx context.Context, newSvc *corev1.Service) (admission.Warnings, error) {
	return nil, nil
}

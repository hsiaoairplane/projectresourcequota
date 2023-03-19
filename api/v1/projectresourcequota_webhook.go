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
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var resourceNameList = []corev1.ResourceName{
	corev1.ResourceCPU,
	corev1.ResourceMemory,
	corev1.ResourceStorage,
	corev1.ResourceEphemeralStorage,
	corev1.ResourcePods,
	corev1.ResourceServices,
	corev1.ResourceReplicationControllers,
	corev1.ResourceQuotas,
	corev1.ResourceSecrets,
	corev1.ResourceConfigMaps,
	corev1.ResourcePersistentVolumeClaims,
	corev1.ResourceServicesNodePorts,
	corev1.ResourceServicesLoadBalancers,
	corev1.ResourceRequestsCPU,
	corev1.ResourceRequestsMemory,
	corev1.ResourceRequestsStorage,
	corev1.ResourceRequestsEphemeralStorage,
	corev1.ResourceLimitsCPU,
	corev1.ResourceLimitsMemory,
	corev1.ResourceLimitsEphemeralStorage,
}

var resourceNameMap = map[corev1.ResourceName]struct{}{
	corev1.ResourceCPU:                      struct{}{},
	corev1.ResourceMemory:                   struct{}{},
	corev1.ResourceStorage:                  struct{}{},
	corev1.ResourceEphemeralStorage:         struct{}{},
	corev1.ResourcePods:                     struct{}{},
	corev1.ResourceServices:                 struct{}{},
	corev1.ResourceReplicationControllers:   struct{}{},
	corev1.ResourceQuotas:                   struct{}{},
	corev1.ResourceSecrets:                  struct{}{},
	corev1.ResourceConfigMaps:               struct{}{},
	corev1.ResourcePersistentVolumeClaims:   struct{}{},
	corev1.ResourceServicesNodePorts:        struct{}{},
	corev1.ResourceServicesLoadBalancers:    struct{}{},
	corev1.ResourceRequestsCPU:              struct{}{},
	corev1.ResourceRequestsMemory:           struct{}{},
	corev1.ResourceRequestsStorage:          struct{}{},
	corev1.ResourceRequestsEphemeralStorage: struct{}{},
	corev1.ResourceLimitsCPU:                struct{}{},
	corev1.ResourceLimitsMemory:             struct{}{},
	corev1.ResourceLimitsEphemeralStorage:   struct{}{},
}

func SetupProjectResourceQuotaWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&ProjectResourceQuota{}).
		WithDefaulter(&projectResourceQuotaAnnotator{mgr.GetClient()}).
		WithValidator(&projectResourceQuotaValidator{mgr.GetClient()}).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-jenting-io-v1-projectresourcequota,mutating=true,failurePolicy=fail,sideEffects=None,groups=jenting.io,resources=projectresourcequotas,verbs=create;update,versions=v1,name=mprojectresourcequota.kb.io,admissionReviewVersions=v1

// projectResourceQuotaAnnotator annotates ProjectResourceQuotas
type projectResourceQuotaAnnotator struct {
	client.Client
}

func (a *projectResourceQuotaAnnotator) Default(ctx context.Context, obj runtime.Object) error {
	return nil
}

//+kubebuilder:webhook:path=/validate-jenting-io-v1-projectresourcequota,mutating=false,failurePolicy=fail,sideEffects=None,groups=jenting.io,resources=projectresourcequotas,verbs=create;update,versions=v1,name=vprojectresourcequota.kb.io,admissionReviewVersions=v1

// projectResourceQuotaValidator validates ProjectResourceQuotas
type projectResourceQuotaValidator struct {
	client.Client
}

// validateNamespace validates the spec.namespaces is not in other CRs
func (v *projectResourceQuotaValidator) validateNamespace(ctx context.Context, prqName string, prqNamespaces []string) error {
	prqList := &ProjectResourceQuotaList{}
	if err := v.Client.List(ctx, prqList); err != nil {
		return err
	}

	for _, prq := range prqList.Items {
		// validate the other projectresourcequota CRs only
		if prqName != prq.Name {
			for _, namespace := range prq.Spec.Namespaces {
				for _, prqNamepsace := range prqNamespaces {
					if namespace == prqNamepsace {
						return fmt.Errorf("namespace %s is already in project %s", prqNamepsace, prq.Name)
					}
				}
			}
		}
	}
	return nil
}

// validateResourceName validates the given resource name is supported
func (v *projectResourceQuotaValidator) validateResourceName(ctx context.Context, rl corev1.ResourceList) error {
	for resourceName := range rl {
		if _, found := resourceNameMap[resourceName]; !found {
			return fmt.Errorf("resource name %s is not supported", resourceName)
		}
	}
	return nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (v *projectResourceQuotaValidator) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	prq, ok := obj.(*ProjectResourceQuota)
	if !ok {
		return fmt.Errorf("expected a ProjectResourceQuota but got a %T", obj)
	}

	// validate the spec.namespaces is not in other CRs
	if err := v.validateNamespace(ctx, prq.Name, prq.Spec.Namespaces); err != nil {
		return err
	}

	// validate the given resource name is supported
	return v.validateResourceName(ctx, prq.Spec.Hard)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (v *projectResourceQuotaValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	log := logf.FromContext(ctx)

	oldPrq, ok := oldObj.(*ProjectResourceQuota)
	if !ok {
		return fmt.Errorf("expected a ProjectResourceQuota but got a %T", oldObj)
	}

	newPrq, ok := newObj.(*ProjectResourceQuota)
	if !ok {
		return fmt.Errorf("expected a ProjectResourceQuota but got a %T", newObj)
	}

	// handle the namespapce removal from the spec.namespaces
	oldNamespaces := sets.NewString(oldPrq.Spec.Namespaces...)
	newNamespaces := sets.NewString(newPrq.Spec.Namespaces...)
	removedNamespaces := oldNamespaces.Difference(newNamespaces)

	log.Info("validate update", "oldNamespaces", oldNamespaces, "newNamespaces", newNamespaces, "removedNamespace", removedNamespaces)

	if len(removedNamespaces) > 0 {
		for _, removedNamespace := range removedNamespaces.List() {
			log.Info("validate update", "newPrq.Name", newPrq.Name, "removedNamespace", removedNamespace)

			listOptions := (&client.ListOptions{}).ApplyOptions([]client.ListOption{
				client.MatchingLabels{ProjectResourceQuotaLabel: newPrq.Name},
				client.InNamespace(removedNamespace),
			})

			// remove the labels from the configmaps.
			cmList := &corev1.ConfigMapList{}
			if err := v.Client.List(ctx, cmList, listOptions); err != nil {
				return err
			}
			for _, cm := range cmList.Items {
				delete(cm.Labels, ProjectResourceQuotaLabel)
				if err := v.Client.Update(ctx, &cm); err != nil {
					return err
				}

				newCM := &corev1.ConfigMap{}
				if err := v.Client.Get(ctx, types.NamespacedName{Namespace: cm.Namespace, Name: cm.Name}, newCM); err != nil {
					return err
				}
				log.Info("validate update", "newCM.Labels", newCM.Labels)
			}

			// remove the labels from the persistentvolumeclaims.
			pvcList := &corev1.PersistentVolumeClaimList{}
			if err := v.Client.List(ctx, pvcList, listOptions); err != nil {
				return err
			}
			for _, pvc := range pvcList.Items {
				delete(pvc.Labels, ProjectResourceQuotaLabel)
				if err := v.Client.Update(ctx, &pvc); err != nil {
					return err
				}
			}

			// remove the labels from the pods
			podList := &corev1.PodList{}
			if err := v.Client.List(ctx, podList, listOptions); err != nil {
				return err
			}
			for _, pod := range podList.Items {
				delete(pod.Labels, ProjectResourceQuotaLabel)
				if err := v.Client.Update(ctx, &pod); err != nil {
					return err
				}
			}

			// remove the labels from the replicationcontrollers
			rcList := &corev1.ReplicationControllerList{}
			if err := v.Client.List(ctx, rcList, listOptions); err != nil {
				return err
			}
			for _, rc := range rcList.Items {
				delete(rc.Labels, ProjectResourceQuotaLabel)
				if err := v.Client.Update(ctx, &rc); err != nil {
					return err
				}
			}

			// remove the labels from the resourcequotas
			rqList := &corev1.ResourceQuotaList{}
			if err := v.Client.List(ctx, rqList, listOptions); err != nil {
				return err
			}
			for _, rq := range rqList.Items {
				delete(rq.Labels, ProjectResourceQuotaLabel)
				if err := v.Client.Update(ctx, &rq); err != nil {
					return err
				}
			}

			// remove the labels from the secrets
			secretList := &corev1.SecretList{}
			if err := v.Client.List(ctx, secretList, listOptions); err != nil {
				return err
			}
			for _, secret := range secretList.Items {
				delete(secret.Labels, ProjectResourceQuotaLabel)
				if err := v.Client.Update(ctx, &secret); err != nil {
					return err
				}
			}

			// remove the labels from the services
			serviceList := &corev1.ServiceList{}
			if err := v.Client.List(ctx, serviceList, listOptions); err != nil {
				return err
			}
			for _, service := range serviceList.Items {
				delete(service.Labels, ProjectResourceQuotaLabel)
				if err := v.Client.Update(ctx, &service); err != nil {
					return err
				}
			}
		}
	}

	// validate the spec.namespaces is not in other CRs
	if err := v.validateNamespace(ctx, newPrq.Name, newPrq.Spec.Namespaces); err != nil {
		return err
	}

	// validate the given resource name is supported
	if err := v.validateResourceName(ctx, newPrq.Spec.Hard); err != nil {
		return err
	}

	// validates the spec.hard is not less than status.used
	for _, resourceName := range resourceNameList {
		hard := newPrq.Spec.Hard[resourceName]
		used := newPrq.Status.Used[resourceName]
		if hard.Cmp(used) == -1 {
			return fmt.Errorf("hard limit %s is less than used %s", hard.String(), used.String())
		}
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (v *projectResourceQuotaValidator) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}

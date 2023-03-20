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

package controller

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	jentingiov1 "github.com/jenting/projectresourcequota/api/v1"
)

// ProjectResourceQuotaReconciler reconciles a ProjectResourceQuota object
type ProjectResourceQuotaReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// removeLabelFromObject removes the label project-resource-quota from the configmaps/persistentvolumeclaims/pods/replicationcontrollers/resourcequotas/secrets/services.
func (r *ProjectResourceQuotaReconciler) removeLabelFromObjects(ctx context.Context, log logr.Logger, prqName string, removedNamespaces sets.String) error {
	if len(removedNamespaces) == 0 {
		return nil
	}

	for _, removedNamespace := range removedNamespaces.List() {
		log.Info("Reconcile ProjectResourceQuota", "prqName", prqName, "removedNamespace", removedNamespace)

		listOptions := (&client.ListOptions{}).ApplyOptions([]client.ListOption{
			client.MatchingLabels{jentingiov1.ProjectResourceQuotaLabel: prqName},
			client.InNamespace(removedNamespace),
		})

		// remove the labels from the configmaps.
		cmList := &corev1.ConfigMapList{}
		if err := r.Client.List(ctx, cmList, listOptions); err != nil {
			log.Error(err, "failed to list configmaps")
			return err
		}
		for _, cm := range cmList.Items {
			delete(cm.Labels, jentingiov1.ProjectResourceQuotaLabel)
			if err := r.Client.Update(ctx, &cm); err != nil {
				log.Error(err, "failed to update configmap %s/%s", cm.Name, cm.Namespace)
				return err
			}
		}

		// remove the labels from the persistentvolumeclaims.
		pvcList := &corev1.PersistentVolumeClaimList{}
		if err := r.Client.List(ctx, pvcList, listOptions); err != nil {
			log.Error(err, "failed to list persistentvolumeclaims")
			return err
		}
		for _, pvc := range pvcList.Items {
			delete(pvc.Labels, jentingiov1.ProjectResourceQuotaLabel)
			if err := r.Client.Update(ctx, &pvc); err != nil {
				log.Error(err, "failed to update persistentvolumeclaim %s/%s", pvc.Name, pvc.Namespace)
				return err
			}
		}

		// remove the labels from the pods
		podList := &corev1.PodList{}
		if err := r.Client.List(ctx, podList, listOptions); err != nil {
			log.Error(err, "failed to list pods")
			return err
		}
		for _, pod := range podList.Items {
			delete(pod.Labels, jentingiov1.ProjectResourceQuotaLabel)
			if err := r.Client.Update(ctx, &pod); err != nil {
				log.Error(err, "failed to update pod %s/%s", pod.Name, pod.Namespace)
				return err
			}
		}

		// remove the labels from the replicationcontrollers
		rcList := &corev1.ReplicationControllerList{}
		if err := r.Client.List(ctx, rcList, listOptions); err != nil {
			log.Error(err, "failed to list replicationcontrollers")
			return err
		}
		for _, rc := range rcList.Items {
			delete(rc.Labels, jentingiov1.ProjectResourceQuotaLabel)
			if err := r.Client.Update(ctx, &rc); err != nil {
				log.Error(err, "failed to update replicationcontroller %s/%s", rc.Name, rc.Namespace)
				return err
			}
		}

		// remove the labels from the resourcequotas
		rqList := &corev1.ResourceQuotaList{}
		if err := r.Client.List(ctx, rqList, listOptions); err != nil {
			log.Error(err, "failed to list resourcequotas")
			return err
		}
		for _, rq := range rqList.Items {
			delete(rq.Labels, jentingiov1.ProjectResourceQuotaLabel)
			if err := r.Client.Update(ctx, &rq); err != nil {
				log.Error(err, "failed to update resourcequota %s/%s", rq.Name, rq.Namespace)
				return err
			}
		}

		// remove the labels from the secrets
		secretList := &corev1.SecretList{}
		if err := r.Client.List(ctx, secretList, listOptions); err != nil {
			log.Error(err, "failed to list secrets")
			return err
		}
		for _, secret := range secretList.Items {
			delete(secret.Labels, jentingiov1.ProjectResourceQuotaLabel)
			if err := r.Client.Update(ctx, &secret); err != nil {
				log.Error(err, "failed to update secret %s/%s", secret.Name, secret.Namespace)
				return err
			}
		}

		// remove the labels from the services
		svcList := &corev1.ServiceList{}
		if err := r.Client.List(ctx, svcList, listOptions); err != nil {
			log.Error(err, "failed to list services")
			return err
		}
		for _, svc := range svcList.Items {
			delete(svc.Labels, jentingiov1.ProjectResourceQuotaLabel)
			if err := r.Client.Update(ctx, &svc); err != nil {
				log.Error(err, "failed to update service %s/%s", svc.Name, svc.Namespace)
				return err
			}
		}
	}
	return nil
}

//+kubebuilder:rbac:groups=jenting.io,resources=projectresourcequotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=jenting.io,resources=projectresourcequotas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=jenting.io,resources=projectresourcequotas/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="",resources=replicationcontrollers,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="",resources=resourcequotas,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ProjectResourceQuotaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Reconcile ProjectResourceQuota", "name", req.Name, "namespace", req.Namespace)

	prq := &jentingiov1.ProjectResourceQuota{}
	if err := r.Get(ctx, req.NamespacedName, prq); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		log.Error(err, "unable to get projectresourcequota")
		return ctrl.Result{}, err
	}

	if prq.DeletionTimestamp != nil {
		removedNamespaces := sets.NewString(prq.Spec.Namespaces...)
		log.Info("Delete ProjectResourceQuota", "removedNamespace", removedNamespaces)

		if err := r.removeLabelFromObjects(ctx, log, prq.Name, removedNamespaces); err != nil {
			log.Error(err, "failed to remove label from objects")
			return ctrl.Result{}, err
		}

		log.Info("Delete ProjectResourceQuota", "remove finalizer", jentingiov1.ProjectResourceQuotaFinalizer)
		if err := jentingiov1.RemoveFinalizer(jentingiov1.ProjectResourceQuotaFinalizer, prq); err != nil {
			log.Error(err, "failed to remove finalizer")
			return ctrl.Result{}, err
		}

		log.Info("Delete ProjectResourceQuota", "update projectresourcequota resource", prq.Name)
		if err := r.Update(ctx, prq); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if prq.Status.Used == nil {
		prq.Status.Used = make(corev1.ResourceList)
	}

	anno, ok := prq.Annotations[corev1.LastAppliedConfigAnnotation]
	if ok {
		oldPrq := &jentingiov1.ProjectResourceQuota{}
		if err := json.Unmarshal([]byte(anno), oldPrq); err != nil {
			log.Error(err, "failed to unmarshal last applied config annotation")
			return ctrl.Result{}, nil
		}

		// handle the namespapce removal from the spec.namespaces
		oldNamespaces := sets.NewString(oldPrq.Spec.Namespaces...)
		newNamespaces := sets.NewString(prq.Spec.Namespaces...)
		removedNamespaces := oldNamespaces.Difference(newNamespaces)

		log.Info("Reconcile ProjectResourceQuota", "oldNamespaces", oldNamespaces, "newNamespaces", newNamespaces, "removedNamespace", removedNamespaces)

		if err := r.removeLabelFromObjects(ctx, log, prq.Name, removedNamespaces); err != nil {
			return ctrl.Result{}, err
		}
	}

	listOptions := (&client.ListOptions{}).ApplyOptions([]client.ListOption{
		client.MatchingLabels{jentingiov1.ProjectResourceQuotaLabel: prq.Name},
	})

	// ConfigMap
	if _, found := prq.Spec.Hard[corev1.ResourceConfigMaps]; found {
		cmList := &corev1.ConfigMapList{}
		if err := r.Client.List(ctx, cmList, listOptions); err != nil {
			return ctrl.Result{}, err
		}
		prq.Status.Used[corev1.ResourceConfigMaps] = resource.MustParse(fmt.Sprintf("%d", len(cmList.Items)))
	}

	// PersistentVolumeClaim
	if _, found := prq.Spec.Hard[corev1.ResourcePersistentVolumeClaims]; found {
		pvcList := &corev1.PersistentVolumeClaimList{}
		if err := r.Client.List(ctx, pvcList, listOptions); err != nil {
			return ctrl.Result{}, err
		}
		prq.Status.Used[corev1.ResourcePersistentVolumeClaims] = resource.MustParse(fmt.Sprintf("%d", len(pvcList.Items)))
	}

	// Pod
	if _, found := prq.Spec.Hard[corev1.ResourcePods]; found {
		podList := &corev1.PodList{}
		if err := r.Client.List(ctx, podList, listOptions); err != nil {
			return ctrl.Result{}, err
		}
		prq.Status.Used[corev1.ResourcePods] = resource.MustParse(fmt.Sprintf("%d", len(podList.Items)))

		var requestCPU, requestMemory, requestStorage, requestEphemeralStorage resource.Quantity
		var limitCPU, limitMemory, limitEphemeralStorage resource.Quantity
		for _, pod := range podList.Items {
			for _, container := range pod.Spec.Containers {
				requestCPU.Add(container.Resources.Requests[corev1.ResourceCPU])
				requestMemory.Add(container.Resources.Requests[corev1.ResourceMemory])
				requestStorage.Add(container.Resources.Requests[corev1.ResourceStorage])
				requestEphemeralStorage.Add(container.Resources.Requests[corev1.ResourceEphemeralStorage])

				limitCPU.Add(container.Resources.Limits[corev1.ResourceLimitsCPU])
				limitMemory.Add(container.Resources.Limits[corev1.ResourceLimitsMemory])
				limitEphemeralStorage.Add(container.Resources.Limits[corev1.ResourceLimitsEphemeralStorage])
			}
		}

		if _, found := prq.Spec.Hard[corev1.ResourceCPU]; found {
			prq.Status.Used[corev1.ResourceCPU] = requestCPU
		}
		if _, found := prq.Spec.Hard[corev1.ResourceRequestsCPU]; found {
			prq.Status.Used[corev1.ResourceRequestsCPU] = requestCPU
		}
		if _, found := prq.Spec.Hard[corev1.ResourceMemory]; found {
			prq.Status.Used[corev1.ResourceMemory] = requestMemory
		}
		if _, found := prq.Spec.Hard[corev1.ResourceRequestsMemory]; found {
			prq.Status.Used[corev1.ResourceRequestsMemory] = requestMemory
		}
		if _, found := prq.Spec.Hard[corev1.ResourceStorage]; found {
			prq.Status.Used[corev1.ResourceStorage] = requestStorage
		}
		if _, found := prq.Spec.Hard[corev1.ResourceRequestsStorage]; found {
			prq.Status.Used[corev1.ResourceRequestsStorage] = requestStorage
		}
		if _, found := prq.Spec.Hard[corev1.ResourceEphemeralStorage]; found {
			prq.Status.Used[corev1.ResourceEphemeralStorage] = requestEphemeralStorage
		}
		if _, found := prq.Spec.Hard[corev1.ResourceRequestsEphemeralStorage]; found {
			prq.Status.Used[corev1.ResourceRequestsEphemeralStorage] = requestEphemeralStorage
		}

		if _, found := prq.Spec.Hard[corev1.ResourceLimitsCPU]; found {
			prq.Status.Used[corev1.ResourceLimitsCPU] = limitCPU
		}
		if _, found := prq.Spec.Hard[corev1.ResourceLimitsMemory]; found {
			prq.Status.Used[corev1.ResourceLimitsMemory] = limitMemory
		}
		if _, found := prq.Spec.Hard[corev1.ResourceLimitsEphemeralStorage]; found {
			prq.Status.Used[corev1.ResourceLimitsEphemeralStorage] = limitEphemeralStorage
		}
	}

	// ReplicationController
	if _, found := prq.Spec.Hard[corev1.ResourceReplicationControllers]; found {
		rcList := &corev1.ReplicationControllerList{}
		if err := r.Client.List(ctx, rcList, listOptions); err != nil {
			return ctrl.Result{}, err
		}
		prq.Status.Used[corev1.ResourceReplicationControllers] = resource.MustParse(fmt.Sprintf("%d", len(rcList.Items)))
	}

	// ResourceQuota
	if _, found := prq.Spec.Hard[corev1.ResourceQuotas]; found {
		rqList := &corev1.ResourceQuotaList{}
		if err := r.Client.List(ctx, rqList, listOptions); err != nil {
			return ctrl.Result{}, err
		}
		prq.Status.Used[corev1.ResourceQuotas] = resource.MustParse(fmt.Sprintf("%d", len(rqList.Items)))
	}

	// Secret
	if _, found := prq.Spec.Hard[corev1.ResourceSecrets]; found {
		secretList := &corev1.SecretList{}
		if err := r.Client.List(ctx, secretList, listOptions); err != nil {
			return ctrl.Result{}, err
		}
		prq.Status.Used[corev1.ResourceSecrets] = resource.MustParse(fmt.Sprintf("%d", len(secretList.Items)))
	}

	// Service
	if _, found := prq.Spec.Hard[corev1.ResourceServices]; found {
		svcList := &corev1.ServiceList{}
		if err := r.Client.List(ctx, svcList, listOptions); err != nil {
			return ctrl.Result{}, err
		}
		prq.Status.Used[corev1.ResourceServices] = resource.MustParse(fmt.Sprintf("%d", len(svcList.Items)))

		var npCount, lbCount int
		for _, svc := range svcList.Items {
			switch svc.Spec.Type {
			case corev1.ServiceTypeNodePort:
				npCount++
			case corev1.ServiceTypeLoadBalancer:
				lbCount++
			}
		}

		if _, found := prq.Spec.Hard[corev1.ResourceServicesNodePorts]; found {
			prq.Status.Used[corev1.ResourceServicesNodePorts] = resource.MustParse(fmt.Sprintf("%d", npCount))
		}
		if _, found := prq.Spec.Hard[corev1.ResourceServicesLoadBalancers]; found {
			prq.Status.Used[corev1.ResourceServicesLoadBalancers] = resource.MustParse(fmt.Sprintf("%d", lbCount))
		}
	}

	if err := r.Status().Update(ctx, prq); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectResourceQuotaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jentingiov1.ProjectResourceQuota{}, builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Watches(&source.Kind{Type: &corev1.ConfigMap{}}, // ConfigMap
			handler.EnqueueRequestsFromMapFunc(r.findObjects),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(&source.Kind{Type: &corev1.PersistentVolumeClaim{}}, // PersistentVolumeClaim
			handler.EnqueueRequestsFromMapFunc(r.findObjects),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(&source.Kind{Type: &corev1.Pod{}}, // Pod
			handler.EnqueueRequestsFromMapFunc(r.findObjects),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(&source.Kind{Type: &corev1.ReplicationController{}}, // ReplicationController
			handler.EnqueueRequestsFromMapFunc(r.findObjects),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(&source.Kind{Type: &corev1.ResourceQuota{}}, // ResourceQuota
			handler.EnqueueRequestsFromMapFunc(r.findObjects),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(&source.Kind{Type: &corev1.Secret{}}, // Secret
			handler.EnqueueRequestsFromMapFunc(r.findObjects),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(&source.Kind{Type: &corev1.Service{}}, // Service
			handler.EnqueueRequestsFromMapFunc(r.findObjects),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *ProjectResourceQuotaReconciler) findObjects(obj client.Object) []reconcile.Request {
	prqName, found := obj.GetLabels()[jentingiov1.ProjectResourceQuotaLabel]
	if !found {
		return nil
	}
	return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: prqName}}}
}

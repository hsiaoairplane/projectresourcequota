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

// removeAnnotationFromObjects removes the annotation project-resource-quota from the configmaps/persistentvolumeclaims/pods/replicationcontrollers/resourcequotas/secrets/services.
func (r *ProjectResourceQuotaReconciler) removeAnnotationFromObjects(ctx context.Context, log logr.Logger, prqName string, removedNamespaces sets.String) error {
	if len(removedNamespaces) == 0 {
		return nil
	}

	for _, removedNamespace := range removedNamespaces.List() {
		log.Info("Reconcile ProjectResourceQuota", "prqName", prqName, "removedNamespace", removedNamespace)

		// remove the annotations from the configmaps.
		cmList := &corev1.ConfigMapList{}
		if err := r.Client.List(ctx, cmList, &client.ListOptions{Namespace: removedNamespace}); err != nil {
			log.Error(err, "failed to list configmaps")
			return err
		}
		for _, cm := range cmList.Items {
			if !jentingiov1.IsAnnotationExists(&cm, jentingiov1.ProjectResourceQuotaAnnotation) {
				continue
			}

			if err := jentingiov1.RemoveAnnotation(&cm, jentingiov1.ProjectResourceQuotaAnnotation); err != nil {
				log.Error(err, "failed to remove annotation from configmap %s/%s", cm.Name, cm.Namespace)
				return err
			}

			if err := r.Client.Update(ctx, &cm); err != nil {
				log.Error(err, "failed to update configmap %s/%s", cm.Name, cm.Namespace)
				return err
			}
		}

		// remove the annotations from the persistentvolumeclaims.
		pvcList := &corev1.PersistentVolumeClaimList{}
		if err := r.Client.List(ctx, pvcList, &client.ListOptions{Namespace: removedNamespace}); err != nil {
			log.Error(err, "failed to list persistentvolumeclaims")
			return err
		}
		for _, pvc := range pvcList.Items {
			if !jentingiov1.IsAnnotationExists(&pvc, jentingiov1.ProjectResourceQuotaAnnotation) {
				continue
			}

			if err := jentingiov1.RemoveAnnotation(&pvc, jentingiov1.ProjectResourceQuotaAnnotation); err != nil {
				log.Error(err, "failed to remove annotation from persistentvolumeclaim %s/%s", pvc.Name, pvc.Namespace)
				return err
			}

			if err := r.Client.Update(ctx, &pvc); err != nil {
				log.Error(err, "failed to update persistentvolumeclaim %s/%s", pvc.Name, pvc.Namespace)
				return err
			}
		}

		// remove the annotations from the pods
		podList := &corev1.PodList{}
		if err := r.Client.List(ctx, podList, &client.ListOptions{Namespace: removedNamespace}); err != nil {
			log.Error(err, "failed to list pods")
			return err
		}
		for _, pod := range podList.Items {
			if !jentingiov1.IsAnnotationExists(&pod, jentingiov1.ProjectResourceQuotaAnnotation) {
				continue
			}

			if err := jentingiov1.RemoveAnnotation(&pod, jentingiov1.ProjectResourceQuotaAnnotation); err != nil {
				log.Error(err, "failed to remove annotation from pod %s/%s", pod.Name, pod.Namespace)
				return err
			}

			if err := r.Client.Update(ctx, &pod); err != nil {
				log.Error(err, "failed to update pod %s/%s", pod.Name, pod.Namespace)
				return err
			}
		}

		// remove the annotations from the replicationcontrollers
		rcList := &corev1.ReplicationControllerList{}
		if err := r.Client.List(ctx, rcList, &client.ListOptions{Namespace: removedNamespace}); err != nil {
			log.Error(err, "failed to list replicationcontrollers")
			return err
		}
		for _, rc := range rcList.Items {
			if !jentingiov1.IsAnnotationExists(&rc, jentingiov1.ProjectResourceQuotaAnnotation) {
				continue
			}

			if err := jentingiov1.RemoveAnnotation(&rc, jentingiov1.ProjectResourceQuotaAnnotation); err != nil {
				log.Error(err, "failed to remove annotation from replicationcontroller %s/%s", rc.Name, rc.Namespace)
				return err
			}

			if err := r.Client.Update(ctx, &rc); err != nil {
				log.Error(err, "failed to update replicationcontroller %s/%s", rc.Name, rc.Namespace)
				return err
			}
		}

		// remove the annotations from the resourcequotas
		rqList := &corev1.ResourceQuotaList{}
		if err := r.Client.List(ctx, rqList, &client.ListOptions{Namespace: removedNamespace}); err != nil {
			log.Error(err, "failed to list resourcequotas")
			return err
		}
		for _, rq := range rqList.Items {
			if !jentingiov1.IsAnnotationExists(&rq, jentingiov1.ProjectResourceQuotaAnnotation) {
				continue
			}

			if err := jentingiov1.RemoveAnnotation(&rq, jentingiov1.ProjectResourceQuotaAnnotation); err != nil {
				log.Error(err, "failed to remove annotation from resourcequota %s/%s", rq.Name, rq.Namespace)
				return err
			}

			if err := r.Client.Update(ctx, &rq); err != nil {
				log.Error(err, "failed to update resourcequota %s/%s", rq.Name, rq.Namespace)
				return err
			}
		}

		// remove the annotations from the secrets
		secretList := &corev1.SecretList{}
		if err := r.Client.List(ctx, secretList, &client.ListOptions{Namespace: removedNamespace}); err != nil {
			log.Error(err, "failed to list secrets")
			return err
		}
		for _, secret := range secretList.Items {
			if !jentingiov1.IsAnnotationExists(&secret, jentingiov1.ProjectResourceQuotaAnnotation) {
				continue
			}

			if err := jentingiov1.RemoveAnnotation(&secret, jentingiov1.ProjectResourceQuotaAnnotation); err != nil {
				log.Error(err, "failed to remove annotation from secret %s/%s", secret.Name, secret.Namespace)
				return err
			}

			if err := r.Client.Update(ctx, &secret); err != nil {
				log.Error(err, "failed to update secret %s/%s", secret.Name, secret.Namespace)
				return err
			}
		}

		// remove the annotations from the services
		svcList := &corev1.ServiceList{}
		if err := r.Client.List(ctx, svcList, &client.ListOptions{Namespace: removedNamespace}); err != nil {
			log.Error(err, "failed to list services")
			return err
		}
		for _, svc := range svcList.Items {
			if !jentingiov1.IsAnnotationExists(&svc, jentingiov1.ProjectResourceQuotaAnnotation) {
				continue
			}

			if err := jentingiov1.RemoveAnnotation(&svc, jentingiov1.ProjectResourceQuotaAnnotation); err != nil {
				log.Error(err, "failed to remove annotation from service %s/%s", svc.Name, svc.Namespace)
				return err
			}

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

	// the projectresourcequota is under deletion
	if prq.DeletionTimestamp != nil {
		removedNamespaces := sets.NewString(prq.Spec.Namespaces...)
		log.Info("Delete ProjectResourceQuota", "removedNamespace", removedNamespaces)

		if err := r.removeAnnotationFromObjects(ctx, log, prq.Name, removedNamespaces); err != nil {
			log.Error(err, "failed to remove annotation from objects")
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

	// handle the projectresourcequota spec.namespaces removal
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
		if err := r.removeAnnotationFromObjects(ctx, log, prq.Name, removedNamespaces); err != nil {
			return ctrl.Result{}, err
		}
	}

	if prq.Status.Used == nil {
		prq.Status.Used = make(corev1.ResourceList)
	} else {
		// clean up the status.used before calculation
		for k := range prq.Status.Used {
			delete(prq.Status.Used, k)
		}
	}

	// calculate the current used resources within the project (across multiple namespaces)
	for _, namespace := range prq.Spec.Namespaces {
		// ConfigMap
		if _, found := prq.Spec.Hard[corev1.ResourceConfigMaps]; found {
			cmList := &corev1.ConfigMapList{}
			if err := r.Client.List(ctx, cmList, &client.ListOptions{Namespace: namespace}); err != nil {
				return ctrl.Result{}, err
			}

			var count int
			for _, cm := range cmList.Items {
				if jentingiov1.IsAnnotationExists(&cm, jentingiov1.ProjectResourceQuotaAnnotation) {
					count++
				}
			}

			used := prq.Status.Used[corev1.ResourceConfigMaps]
			used.Add(resource.MustParse(fmt.Sprintf("%d", count)))
			prq.Status.Used[corev1.ResourceConfigMaps] = used
		}

		// PersistentVolumeClaim
		if _, found := prq.Spec.Hard[corev1.ResourcePersistentVolumeClaims]; found {
			pvcList := &corev1.PersistentVolumeClaimList{}
			if err := r.Client.List(ctx, pvcList, &client.ListOptions{Namespace: namespace}); err != nil {
				return ctrl.Result{}, err
			}

			var count int
			for _, pvc := range pvcList.Items {
				if jentingiov1.IsAnnotationExists(&pvc, jentingiov1.ProjectResourceQuotaAnnotation) {
					count++
				}
			}

			used := prq.Status.Used[corev1.ResourcePersistentVolumeClaims]
			used.Add(resource.MustParse(fmt.Sprintf("%d", count)))
			prq.Status.Used[corev1.ResourcePersistentVolumeClaims] = used
		}

		// Pod
		if _, found := prq.Spec.Hard[corev1.ResourcePods]; found {
			podList := &corev1.PodList{}
			if err := r.Client.List(ctx, podList, &client.ListOptions{Namespace: namespace}); err != nil {
				return ctrl.Result{}, err
			}

			var count int
			for _, pvc := range podList.Items {
				if jentingiov1.IsAnnotationExists(&pvc, jentingiov1.ProjectResourceQuotaAnnotation) {
					count++
				}
			}
			used := prq.Status.Used[corev1.ResourcePods]
			used.Add(resource.MustParse(fmt.Sprintf("%d", count)))
			prq.Status.Used[corev1.ResourcePods] = used

			var requestCPU, requestMemory, requestStorage, requestEphemeralStorage resource.Quantity
			var limitCPU, limitMemory, limitEphemeralStorage resource.Quantity
			for _, pod := range podList.Items {
				for _, container := range pod.Spec.Containers {
					// requests
					quality := container.Resources.Requests.Cpu()
					if quality != nil {
						requestCPU.Add(*quality)
					}
					quality = container.Resources.Requests.Memory()
					if quality != nil {
						requestMemory.Add(*quality)
					}
					quality = container.Resources.Requests.Storage()
					if quality != nil {
						requestStorage.Add(*quality)
					}
					quality = container.Resources.Requests.StorageEphemeral()
					if quality != nil {
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
			}

			if _, found := prq.Spec.Hard[corev1.ResourceCPU]; found {
				used := prq.Status.Used[corev1.ResourceCPU]
				used.Add(requestCPU)
				prq.Status.Used[corev1.ResourceCPU] = used
			}
			if _, found := prq.Spec.Hard[corev1.ResourceRequestsCPU]; found {
				used := prq.Status.Used[corev1.ResourceRequestsCPU]
				used.Add(requestCPU)
				prq.Status.Used[corev1.ResourceRequestsCPU] = used
			}
			if _, found := prq.Spec.Hard[corev1.ResourceMemory]; found {
				used := prq.Status.Used[corev1.ResourceMemory]
				used.Add(requestMemory)
				prq.Status.Used[corev1.ResourceMemory] = used
			}
			if _, found := prq.Spec.Hard[corev1.ResourceRequestsMemory]; found {
				used := prq.Status.Used[corev1.ResourceRequestsMemory]
				used.Add(requestMemory)
				prq.Status.Used[corev1.ResourceRequestsMemory] = used
			}
			if _, found := prq.Spec.Hard[corev1.ResourceStorage]; found {
				used := prq.Status.Used[corev1.ResourceStorage]
				used.Add(requestStorage)
				prq.Status.Used[corev1.ResourceStorage] = used
			}
			if _, found := prq.Spec.Hard[corev1.ResourceRequestsStorage]; found {
				used := prq.Status.Used[corev1.ResourceRequestsStorage]
				used.Add(requestStorage)
				prq.Status.Used[corev1.ResourceRequestsStorage] = used
			}
			if _, found := prq.Spec.Hard[corev1.ResourceEphemeralStorage]; found {
				used := prq.Status.Used[corev1.ResourceEphemeralStorage]
				used.Add(requestEphemeralStorage)
				prq.Status.Used[corev1.ResourceEphemeralStorage] = used
			}
			if _, found := prq.Spec.Hard[corev1.ResourceRequestsEphemeralStorage]; found {
				used := prq.Status.Used[corev1.ResourceRequestsEphemeralStorage]
				used.Add(requestEphemeralStorage)
				prq.Status.Used[corev1.ResourceRequestsEphemeralStorage] = used
			}
			if _, found := prq.Spec.Hard[corev1.ResourceLimitsCPU]; found {
				used := prq.Status.Used[corev1.ResourceLimitsCPU]
				used.Add(limitCPU)
				prq.Status.Used[corev1.ResourceLimitsCPU] = used
			}
			if _, found := prq.Spec.Hard[corev1.ResourceLimitsMemory]; found {
				used := prq.Status.Used[corev1.ResourceLimitsMemory]
				used.Add(limitMemory)
				prq.Status.Used[corev1.ResourceLimitsMemory] = used
			}
			if _, found := prq.Spec.Hard[corev1.ResourceLimitsEphemeralStorage]; found {
				used := prq.Status.Used[corev1.ResourceLimitsEphemeralStorage]
				used.Add(limitEphemeralStorage)
				prq.Status.Used[corev1.ResourceLimitsEphemeralStorage] = used
			}
		}

		// ReplicationController
		if _, found := prq.Spec.Hard[corev1.ResourceReplicationControllers]; found {
			rcList := &corev1.ReplicationControllerList{}
			if err := r.Client.List(ctx, rcList, &client.ListOptions{Namespace: namespace}); err != nil {
				return ctrl.Result{}, err
			}

			var count int
			for _, pvc := range rcList.Items {
				if jentingiov1.IsAnnotationExists(&pvc, jentingiov1.ProjectResourceQuotaAnnotation) {
					count++
				}
			}
			used := prq.Status.Used[corev1.ResourceReplicationControllers]
			used.Add(resource.MustParse(fmt.Sprintf("%d", count)))
			prq.Status.Used[corev1.ResourceReplicationControllers] = used
		}

		// ResourceQuota
		if _, found := prq.Spec.Hard[corev1.ResourceQuotas]; found {
			rqList := &corev1.ResourceQuotaList{}
			if err := r.Client.List(ctx, rqList, &client.ListOptions{Namespace: namespace}); err != nil {
				return ctrl.Result{}, err
			}

			var count int
			for _, rq := range rqList.Items {
				if jentingiov1.IsAnnotationExists(&rq, jentingiov1.ProjectResourceQuotaAnnotation) {
					count++
				}
			}
			used := prq.Status.Used[corev1.ResourceQuotas]
			used.Add(resource.MustParse(fmt.Sprintf("%d", count)))
			prq.Status.Used[corev1.ResourceQuotas] = used
		}

		// Secret
		if _, found := prq.Spec.Hard[corev1.ResourceSecrets]; found {
			secretList := &corev1.SecretList{}
			if err := r.Client.List(ctx, secretList, &client.ListOptions{Namespace: namespace}); err != nil {
				return ctrl.Result{}, err
			}

			var count int
			for _, secret := range secretList.Items {
				if jentingiov1.IsAnnotationExists(&secret, jentingiov1.ProjectResourceQuotaAnnotation) {
					count++
				}
			}
			used := prq.Status.Used[corev1.ResourceSecrets]
			used.Add(resource.MustParse(fmt.Sprintf("%d", count)))
			prq.Status.Used[corev1.ResourceSecrets] = used
		}

		// Service
		if _, found := prq.Spec.Hard[corev1.ResourceServices]; found {
			svcList := &corev1.ServiceList{}
			if err := r.Client.List(ctx, svcList, &client.ListOptions{Namespace: namespace}); err != nil {
				return ctrl.Result{}, err
			}

			var count, npCount, lbCount int
			for _, svc := range svcList.Items {
				if jentingiov1.IsAnnotationExists(&svc, jentingiov1.ProjectResourceQuotaAnnotation) {
					count++

					switch svc.Spec.Type {
					case corev1.ServiceTypeNodePort:
						npCount++
					case corev1.ServiceTypeLoadBalancer:
						lbCount++
					}
				}
			}
			used := prq.Status.Used[corev1.ResourceServices]
			used.Add(resource.MustParse(fmt.Sprintf("%d", count)))
			prq.Status.Used[corev1.ResourceServices] = used

			if _, found := prq.Spec.Hard[corev1.ResourceServicesNodePorts]; found {
				used := prq.Status.Used[corev1.ResourceServicesNodePorts]
				used.Add(resource.MustParse(fmt.Sprintf("%d", npCount)))
				prq.Status.Used[corev1.ResourceServicesNodePorts] = used
			}
			if _, found := prq.Spec.Hard[corev1.ResourceServicesLoadBalancers]; found {
				used := prq.Status.Used[corev1.ResourceServicesLoadBalancers]
				used.Add(resource.MustParse(fmt.Sprintf("%d", lbCount)))
				prq.Status.Used[corev1.ResourceServicesLoadBalancers] = used
			}
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

func (r *ProjectResourceQuotaReconciler) findObjects(ctx context.Context, obj client.Object) []reconcile.Request {
	prqName, found := obj.GetAnnotations()[jentingiov1.ProjectResourceQuotaAnnotation]
	if !found {
		return nil
	}
	return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: prqName}}}
}

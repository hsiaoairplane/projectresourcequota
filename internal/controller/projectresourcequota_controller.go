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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	jentingiov1 "github.com/jenting/projectresourcequota/api/v1"
)

// ProjectResourceQuotaReconciler reconciles a ProjectResourceQuota object
type ProjectResourceQuotaReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=jenting.io,resources=projectresourcequotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=jenting.io,resources=projectresourcequotas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=jenting.io,resources=projectresourcequotas/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=replicationcontrollers,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=resourcequotas,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ProjectResourceQuotaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Reconcile ProjectResourceQuota", "name", req.Name, "namespace", req.Namespace)

	rs := &jentingiov1.ProjectResourceQuota{}
	if err := r.Get(ctx, req.NamespacedName, rs); err != nil {
		return ctrl.Result{}, err
	}
	if rs.Status.Used == nil {
		rs.Status.Used = make(corev1.ResourceList)
	}

	listOptions := (&client.ListOptions{}).ApplyOptions([]client.ListOption{
		client.MatchingLabels{jentingiov1.ProjectResourceQuotaLabel: rs.Name},
	})

	// ConfigMap
	cmList := &corev1.ConfigMapList{}
	if err := r.Client.List(ctx, cmList, listOptions); err != nil {
		return ctrl.Result{}, err
	}
	rs.Status.Used[corev1.ResourceConfigMaps] = resource.MustParse(fmt.Sprintf("%d", len(cmList.Items)))

	// PersistentVolumeClaim
	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := r.Client.List(ctx, pvcList, listOptions); err != nil {
		return ctrl.Result{}, err
	}
	rs.Status.Used[corev1.ResourcePersistentVolumeClaims] = resource.MustParse(fmt.Sprintf("%d", len(pvcList.Items)))

	// Pod
	podList := &corev1.PodList{}
	if err := r.Client.List(ctx, podList, listOptions); err != nil {
		return ctrl.Result{}, err
	}
	rs.Status.Used[corev1.ResourcePods] = resource.MustParse(fmt.Sprintf("%d", len(podList.Items)))

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

	rs.Status.Used[corev1.ResourceCPU] = requestCPU
	rs.Status.Used[corev1.ResourceRequestsCPU] = requestCPU
	rs.Status.Used[corev1.ResourceMemory] = requestMemory
	rs.Status.Used[corev1.ResourceRequestsMemory] = requestMemory
	rs.Status.Used[corev1.ResourceStorage] = requestStorage
	rs.Status.Used[corev1.ResourceRequestsStorage] = requestStorage
	rs.Status.Used[corev1.ResourceEphemeralStorage] = requestEphemeralStorage
	rs.Status.Used[corev1.ResourceRequestsEphemeralStorage] = requestEphemeralStorage

	rs.Status.Used[corev1.ResourceLimitsCPU] = limitCPU
	rs.Status.Used[corev1.ResourceLimitsMemory] = limitMemory
	rs.Status.Used[corev1.ResourceLimitsEphemeralStorage] = limitEphemeralStorage

	// ReplicationController
	rcList := &corev1.ReplicationControllerList{}
	if err := r.Client.List(ctx, rcList, listOptions); err != nil {
		return ctrl.Result{}, err
	}
	rs.Status.Used[corev1.ResourceReplicationControllers] = resource.MustParse(fmt.Sprintf("%d", len(rcList.Items)))

	// ResourceQuota
	rqList := &corev1.ResourceQuotaList{}
	if err := r.Client.List(ctx, rqList, listOptions); err != nil {
		return ctrl.Result{}, err
	}
	rs.Status.Used[corev1.ResourceQuotas] = resource.MustParse(fmt.Sprintf("%d", len(rqList.Items)))

	// Secret
	secretList := &corev1.SecretList{}
	if err := r.Client.List(ctx, secretList, listOptions); err != nil {
		return ctrl.Result{}, err
	}
	rs.Status.Used[corev1.ResourceSecrets] = resource.MustParse(fmt.Sprintf("%d", len(secretList.Items)))

	// Service
	svcList := &corev1.ServiceList{}
	if err := r.Client.List(ctx, svcList, listOptions); err != nil {
		return ctrl.Result{}, err
	}
	rs.Status.Used[corev1.ResourceServices] = resource.MustParse(fmt.Sprintf("%d", len(svcList.Items)))

	var npCount, lbCount int
	for _, svc := range svcList.Items {
		switch svc.Spec.Type {
		case corev1.ServiceTypeNodePort:
			npCount++
		case corev1.ServiceTypeLoadBalancer:
			lbCount++
		}
	}
	rs.Status.Used[corev1.ResourceServicesNodePorts] = resource.MustParse(fmt.Sprintf("%d", npCount))
	rs.Status.Used[corev1.ResourceServicesLoadBalancers] = resource.MustParse(fmt.Sprintf("%d", lbCount))

	if err := r.Status().Update(ctx, rs); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectResourceQuotaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&jentingiov1.ProjectResourceQuota{}).
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
	log := log.FromContext(context.TODO())

	log.Info("Find objects", "name", obj.GetName(), "namespace", obj.GetNamespace())

	prqName, found := obj.GetLabels()[jentingiov1.ProjectResourceQuotaLabel]
	if !found {
		return nil
	}
	return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: prqName}}}
}

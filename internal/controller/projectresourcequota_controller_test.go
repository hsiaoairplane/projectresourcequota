/*

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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	jentingiov1 "github.com/jenting/projectresourcequota/api/v1"
)

func TestAddResourceList(t *testing.T) {
	dst := corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m")}
	addResourceList(dst, corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("200m"),
		corev1.ResourceMemory: resource.MustParse("128Mi"),
	})

	if got := dst[corev1.ResourceCPU]; got.Cmp(resource.MustParse("300m")) != 0 {
		t.Errorf("cpu = %s, want 300m", got.String())
	}
	if got := dst[corev1.ResourceMemory]; got.Cmp(resource.MustParse("128Mi")) != 0 {
		t.Errorf("memory = %s, want 128Mi", got.String())
	}
}

var _ = Describe("ProjectResourceQuota controller", func() {
	var (
		ctx        context.Context
		reconciler *ProjectResourceQuotaReconciler
	)

	BeforeEach(func() {
		ctx = context.Background()
		reconciler = &ProjectResourceQuotaReconciler{Client: k8sClient, Scheme: scheme.Scheme}
	})

	reconcile := func(name string) {
		_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: name}})
		Expect(err).NotTo(HaveOccurred())
	}

	newNamespace := func(name string) {
		Expect(k8sClient.Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		})).To(Succeed())
	}

	annotated := func(prqName string) map[string]string {
		return map[string]string{jentingiov1.ProjectResourceQuotaAnnotation: prqName}
	}

	It("counts annotated objects and sums pod effective requests", func() {
		newNamespace("team-a")

		prq := &jentingiov1.ProjectResourceQuota{
			ObjectMeta: metav1.ObjectMeta{Name: "prq-a"},
			Spec: jentingiov1.ProjectResourceQuotaSpec{
				Namespaces: []string{"team-a"},
				Hard: corev1.ResourceList{
					corev1.ResourceConfigMaps:  resource.MustParse("10"),
					corev1.ResourcePods:        resource.MustParse("10"),
					corev1.ResourceRequestsCPU: resource.MustParse("10"),
				},
			},
		}
		Expect(k8sClient.Create(ctx, prq)).To(Succeed())

		// two annotated configmaps, one un-annotated
		Expect(k8sClient.Create(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Name: "cm-1", Namespace: "team-a", Annotations: annotated("prq-a")}})).To(Succeed())
		Expect(k8sClient.Create(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Name: "cm-2", Namespace: "team-a", Annotations: annotated("prq-a")}})).To(Succeed())
		Expect(k8sClient.Create(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Name: "cm-other", Namespace: "team-a"}})).To(Succeed())

		// one annotated pod whose init container request exceeds its regular containers
		Expect(k8sClient.Create(ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "team-a", Annotations: annotated("prq-a")},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{{
					Name: "init", Image: "busybox",
					Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("500m"),
					}},
				}},
				Containers: []corev1.Container{{
					Name: "main", Image: "busybox",
					Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("100m"),
					}},
				}},
			},
		})).To(Succeed())

		reconcile("prq-a")

		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "prq-a"}, prq)).To(Succeed())
		cm := prq.Status.Used[corev1.ResourceConfigMaps]
		Expect(cm.Cmp(resource.MustParse("2"))).To(Equal(0))
		pods := prq.Status.Used[corev1.ResourcePods]
		Expect(pods.Cmp(resource.MustParse("1"))).To(Equal(0))
		// effective requests.cpu = max(sum(containers)=100m, init=500m) = 500m
		cpu := prq.Status.Used[corev1.ResourceRequestsCPU]
		Expect(cpu.Cmp(resource.MustParse("500m"))).To(Equal(0))
		// status records the namespaces it reconciled
		Expect(prq.Status.Namespaces).To(Equal([]string{"team-a"}))
	})

	It("removes the annotation when a namespace leaves spec.namespaces", func() {
		newNamespace("keep")
		newNamespace("drop")

		prq := &jentingiov1.ProjectResourceQuota{
			ObjectMeta: metav1.ObjectMeta{Name: "prq-b"},
			Spec: jentingiov1.ProjectResourceQuotaSpec{
				Namespaces: []string{"keep", "drop"},
				Hard:       corev1.ResourceList{corev1.ResourceConfigMaps: resource.MustParse("10")},
			},
		}
		Expect(k8sClient.Create(ctx, prq)).To(Succeed())

		cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Name: "cm-drop", Namespace: "drop", Annotations: annotated("prq-b")}}
		Expect(k8sClient.Create(ctx, cm)).To(Succeed())

		// first reconcile records status.namespaces = [keep, drop]
		reconcile("prq-b")
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "prq-b"}, prq)).To(Succeed())
		Expect(prq.Status.Namespaces).To(ConsistOf("keep", "drop"))

		// drop the "drop" namespace from spec
		prq.Spec.Namespaces = []string{"keep"}
		Expect(k8sClient.Update(ctx, prq)).To(Succeed())

		// second reconcile must strip the annotation from objects in "drop"
		reconcile("prq-b")

		updated := &corev1.ConfigMap{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "cm-drop", Namespace: "drop"}, updated)).To(Succeed())
		Expect(jentingiov1.IsAnnotationExists(updated, jentingiov1.ProjectResourceQuotaAnnotation)).To(BeFalse())
	})
})

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

package v1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("ProjectResourceQuota webhook", func() {
	newPRQ := func(name string, hard corev1.ResourceList, namespaces ...string) *ProjectResourceQuota {
		return &ProjectResourceQuota{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: ProjectResourceQuotaSpec{
				Namespaces: namespaces,
				Hard:       hard,
			},
		}
	}

	It("adds the finalizer on create", func() {
		prq := newPRQ("wh-finalizer", corev1.ResourceList{corev1.ResourceConfigMaps: resource.MustParse("1")}, "wh-ns-finalizer")
		Expect(k8sClient.Create(ctx, prq)).To(Succeed())

		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "wh-finalizer"}, prq)).To(Succeed())
		Expect(prq.Finalizers).To(ContainElement(ProjectResourceQuotaFinalizer))
	})

	It("rejects an unsupported resource name", func() {
		prq := newPRQ("wh-badname", corev1.ResourceList{
			corev1.ResourceName("jenting.io/unsupported"): resource.MustParse("1"),
		}, "wh-ns-badname")
		err := k8sClient.Create(ctx, prq)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not supported"))
	})

	It("rejects a namespace already used by another project", func() {
		Expect(k8sClient.Create(ctx, newPRQ("wh-conflict-1",
			corev1.ResourceList{corev1.ResourceConfigMaps: resource.MustParse("1")}, "wh-shared"))).To(Succeed())

		// the webhook validates against its cached client; retry until the
		// previously created project is observed and the conflict is reported.
		Eventually(func() string {
			conflict := newPRQ("", corev1.ResourceList{corev1.ResourceConfigMaps: resource.MustParse("1")}, "wh-shared")
			conflict.GenerateName = "wh-conflict-"
			if err := k8sClient.Create(ctx, conflict); err != nil {
				return err.Error()
			}
			return ""
		}, "5s", "200ms").Should(ContainSubstring("already in project"))
	})

	It("rejects lowering hard below the current used on update", func() {
		prq := newPRQ("wh-hardused", corev1.ResourceList{corev1.ResourceConfigMaps: resource.MustParse("5")}, "wh-ns-hardused")
		Expect(k8sClient.Create(ctx, prq)).To(Succeed())

		// record some usage in the status subresource
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "wh-hardused"}, prq)).To(Succeed())
		prq.Status.Used = corev1.ResourceList{corev1.ResourceConfigMaps: resource.MustParse("3")}
		Expect(k8sClient.Status().Update(ctx, prq)).To(Succeed())

		// lowering the hard limit below the used count must be rejected
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "wh-hardused"}, prq)).To(Succeed())
		prq.Spec.Hard[corev1.ResourceConfigMaps] = resource.MustParse("2")
		err := k8sClient.Update(ctx, prq)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("less than used"))
	})
})

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
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func container(cpu, memory string) corev1.Container {
	return corev1.Container{
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(memory),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(memory),
			},
		},
	}
}

func TestPodEffectiveRequests(t *testing.T) {
	always := corev1.ContainerRestartPolicyAlways

	tests := []struct {
		name       string
		pod        corev1.Pod
		wantCPU    string
		wantMemory string
	}{
		{
			name: "sums regular containers",
			pod: corev1.Pod{Spec: corev1.PodSpec{
				Containers: []corev1.Container{container("100m", "128Mi"), container("200m", "256Mi")},
			}},
			wantCPU:    "300m",
			wantMemory: "384Mi",
		},
		{
			name: "init container larger than the sum wins",
			pod: corev1.Pod{Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{container("500m", "1Gi")},
				Containers:     []corev1.Container{container("100m", "128Mi"), container("200m", "256Mi")},
			}},
			wantCPU:    "500m",
			wantMemory: "1Gi",
		},
		{
			name: "init container smaller than the sum is ignored",
			pod: corev1.Pod{Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{container("100m", "64Mi")},
				Containers:     []corev1.Container{container("200m", "256Mi"), container("200m", "256Mi")},
			}},
			wantCPU:    "400m",
			wantMemory: "512Mi",
		},
		{
			name: "restartable init (sidecar) is added to the sum",
			pod: corev1.Pod{Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{{
					RestartPolicy: &always,
					Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("250m"),
						corev1.ResourceMemory: resource.MustParse("128Mi"),
					}},
				}},
				Containers: []corev1.Container{container("300m", "256Mi")},
			}},
			wantCPU:    "550m",
			wantMemory: "384Mi",
		},
		{
			name: "pod overhead is included",
			pod: corev1.Pod{Spec: corev1.PodSpec{
				Overhead: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				},
				Containers: []corev1.Container{container("300m", "256Mi")},
			}},
			wantCPU:    "400m",
			wantMemory: "320Mi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PodEffectiveRequests(&tt.pod)
			assertQuantity(t, "cpu", got[corev1.ResourceCPU], tt.wantCPU)
			assertQuantity(t, "memory", got[corev1.ResourceMemory], tt.wantMemory)
		})
	}
}

func TestPodEffectiveLimits(t *testing.T) {
	pod := corev1.Pod{Spec: corev1.PodSpec{
		Containers: []corev1.Container{container("100m", "128Mi"), container("200m", "256Mi")},
	}}
	got := PodEffectiveLimits(&pod)
	assertQuantity(t, "cpu", got[corev1.ResourceCPU], "300m")
	assertQuantity(t, "memory", got[corev1.ResourceMemory], "384Mi")
}

func assertQuantity(t *testing.T, name string, got resource.Quantity, want string) {
	t.Helper()
	wantQ := resource.MustParse(want)
	if got.Cmp(wantQ) != 0 {
		t.Errorf("%s = %s, want %s", name, got.String(), want)
	}
}

func TestFinalizer(t *testing.T) {
	obj := &corev1.ConfigMap{}

	if err := AddFinalizer("a", obj); err != nil {
		t.Fatalf("AddFinalizer: %v", err)
	}
	// adding the same finalizer twice must not duplicate it
	if err := AddFinalizer("a", obj); err != nil {
		t.Fatalf("AddFinalizer (again): %v", err)
	}
	if got := obj.GetFinalizers(); len(got) != 1 || got[0] != "a" {
		t.Fatalf("finalizers = %v, want [a]", obj.GetFinalizers())
	}

	if err := RemoveFinalizer("a", obj); err != nil {
		t.Fatalf("RemoveFinalizer: %v", err)
	}
	if got := obj.GetFinalizers(); len(got) != 0 {
		t.Fatalf("finalizers = %v, want empty", got)
	}
}

func TestAnnotation(t *testing.T) {
	obj := &corev1.ConfigMap{}

	if IsAnnotationExists(obj, "k") {
		t.Fatal("annotation should not exist on a fresh object")
	}
	if err := AddAnnotation(obj, "k", "v"); err != nil {
		t.Fatalf("AddAnnotation: %v", err)
	}
	if !IsAnnotationExists(obj, "k") {
		t.Fatal("annotation should exist after AddAnnotation")
	}
	if got := obj.GetAnnotations()["k"]; got != "v" {
		t.Fatalf("annotation value = %q, want v", got)
	}
	if err := RemoveAnnotation(obj, "k"); err != nil {
		t.Fatalf("RemoveAnnotation: %v", err)
	}
	if IsAnnotationExists(obj, "k") {
		t.Fatal("annotation should not exist after RemoveAnnotation")
	}
}

func TestValidateResourceNameSupported(t *testing.T) {
	v := &projectResourceQuotaValidator{}

	supported := corev1.ResourceList{
		corev1.ResourceCPU:               resource.MustParse("1"),
		corev1.ResourceRequestsMemory:    resource.MustParse("1Gi"),
		corev1.ResourceServicesNodePorts: resource.MustParse("2"),
	}
	if err := v.validateResourceName(t.Context(), supported); err != nil {
		t.Errorf("validateResourceName(supported) = %v, want nil", err)
	}

	unsupported := corev1.ResourceList{
		corev1.ResourceName("jenting.io/unsupported"): resource.MustParse("1"),
	}
	if err := v.validateResourceName(t.Context(), unsupported); err == nil {
		t.Error("validateResourceName(unsupported) = nil, want error")
	}
}

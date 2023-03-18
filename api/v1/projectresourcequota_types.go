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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ProjectResourceQuotaSpec defines the desired state of ProjectResourceQuota
type ProjectResourceQuotaSpec struct {
	//+required
	Namespaces []string `json:"namespaces"`
	//+optional
	Hard corev1.ResourceList `json:"hard,omitempty"`
}

// ProjectResourceQuotaStatus defines the observed state of ProjectResourceQuota
type ProjectResourceQuotaStatus struct {
	//+optional
	Used corev1.ResourceList `json:"used,omitempty" protobuf:"bytes,2,rep,name=used,casttype=ResourceList,castkey=ResourceName"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:resource:shortName=prq
//+kubebuilder:printcolumn:name="Namespaces",type="string",JSONPath=".spec.namespaces",description="Namespaces"
//+kubebuilder:printcolumn:name="Hard",type="string",JSONPath=".spec.hard",description="Hard"
//+kubebuilder:printcolumn:name="Used",type="string",JSONPath=".status.used",description="Used"

// ProjectResourceQuota is the Schema for the projectresourcequotas API
type ProjectResourceQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectResourceQuotaSpec   `json:"spec,omitempty"`
	Status ProjectResourceQuotaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProjectResourceQuotaList contains a list of ProjectResourceQuota
type ProjectResourceQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectResourceQuota `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProjectResourceQuota{}, &ProjectResourceQuotaList{})
}

/*
Copyright 2021.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// HelmApp is the Schema for the HelmApp API
// +kubebuilder:resource:shortName=happ
// +kubebuilder:printcolumn:name="phase",type=string,JSONPath=`.status.phase`
type HelmApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   *HelmAppSpec   `json:"spec,omitempty"`
	Status *HelmAppStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HelmAppList contains a list of HelmApp
type HelmAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []*HelmApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HelmApp{}, &HelmAppList{})
}

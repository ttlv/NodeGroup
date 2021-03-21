/*
Copyright 2021 ttlv.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodeGroupSpec defines the desired state of NodeGroup
type NodeGroupSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Selector map[string]string `json:"selector"`
}

// NodeGroupStatus defines the observed state of NodeGroup
type NodeGroupStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	NodeList []NodeList `json:"nodeList"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// NodeGroup is the Schema for the nodegroups API
type NodeGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeGroupSpec   `json:"spec,omitempty"`
	Status NodeGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NodeGroupList contains a list of NodeGroup
type NodeGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeGroup `json:"items"`
}

type NodeList struct {
	NodeName            string `json:"nodeName"`
	NodeMaintenanceName string `json:"nodeMaintenanceName"`
	Ready               string `json:"ready"`
	Maintenanable       string `json:"maintenanable"`
}

func init() {
	SchemeBuilder.Register(&NodeGroup{}, &NodeGroupList{})
}

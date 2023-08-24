/*
Copyright 2023 The Kubernetes crd-controller Authors.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UnitedDeployment is a specification for a UnitedDeployment resource
type UnitedDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"objectMeta,omitempty"`
	Spec              UnitedDeploymentSpec   `json:"spec"`
	Status            UnitedDeploymentStatus `json:"status"`
}

// UnitedDeploymentSpec is the spec for a UnitedDeployment resource
type UnitedDeploymentSpec struct {
	DeploymentName string `json:"deploymentName"`
	Replicas       *int32 `json:"replicas"`
}

// UnitedDeploymentStatus is the status for a UnitedDeployment resource
type UnitedDeploymentStatus struct {
	AvailableReplicas *int32 `json:"availableReplicas"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// UnitedDeploymentList is a list of UnitedDeployment resources
type UnitedDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UnitedDeployment `json:"items"`
}

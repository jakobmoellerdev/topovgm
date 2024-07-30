/*
Copyright 2024.

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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VolumeGroupSpec defines the desired state of VolumeGroup
type VolumeGroupSpec struct {
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="the node cannot be changed once set"
	NodeName string `json:"nodeName"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="the name on the node cannot be changed once set"
	NameOnNode *string `json:"nameOnNode,omitempty"`

	PVs []string `json:"pvs"`

	Devices     []string `json:"devices,omitempty"`
	DevicesFile *string  `json:"devicesFile,omitempty"`

	Tags []string `json:"tags,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="the maximum amount of logical volumes cannot be changed once set"
	MaximumLogicalVolumes *int `json:"maximumLogicalVolumes,omitempty"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="the maximum amount of physical volumes cannot be changed once set"
	MaximumPhysicalVolumes *int `json:"maximumPhysicalVolumes,omitempty"`

	PhysicalExtentSize *resource.Quantity `json:"physicalExtentSize,omitempty"`

	AllocationPolicy *string `json:"allocationPolicy,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="the data alignment cannot be changed once set"
	DataAlignment *resource.Quantity `json:"dataAlignment,omitempty"`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="the data alignment offset cannot be changed once set"
	DataAlignmentOffset *resource.Quantity `json:"dataAlignmentOffset,omitempty"`

	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="zeroing cannot be changed once set"
	Zero           *bool `json:"zero,omitempty"`
	AutoActivation *bool `json:"autoActivation,omitempty"`
}

// VolumeGroupStatus defines the observed state of VolumeGroup
type VolumeGroupStatus struct {
	Name  string `json:"name"`
	UUID  string `json:"uuid,omitempty"`
	SysID string `json:"sysid,omitempty"`

	PVs []string `json:"pvs,omitempty"`

	VGAttributes string   `json:"attr,omitempty"`
	Tags         []string `json:"tags,omitempty"`

	ExtentSize  *resource.Quantity `json:"extentSize,omitempty"`
	ExtentCount int64              `json:"extentCount,omitempty"`

	SeqNo int64 `json:"seqno,omitempty"`

	Size *resource.Quantity `json:"size,omitempty"`
	Free *resource.Quantity `json:"free,omitempty"`

	PvCount        int64 `json:"count,omitempty"`
	MissingPVCount int64 `json:"missingPvCount,omitempty"`
	MaxPv          int64 `json:"maxPv,omitempty"`

	LvCount int64 `json:"lvCount,omitempty"`
	MaxLv   int64 `json:"maxLv,omitempty"`

	SnapCount int64 `json:"snapCount,omitempty"`

	MDACount     int64 `json:"mdaCount,omitempty"`
	MDAUsedCount int64 `json:"mdaUsedCount,omitempty"`
	MDACopies    int64 `json:"mdaCopies,omitempty"`

	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VolumeGroup is the Schema for the volumegroups API
type VolumeGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VolumeGroupSpec   `json:"spec,omitempty"`
	Status VolumeGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VolumeGroupList contains a list of VolumeGroup
type VolumeGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VolumeGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VolumeGroup{}, &VolumeGroupList{})
}

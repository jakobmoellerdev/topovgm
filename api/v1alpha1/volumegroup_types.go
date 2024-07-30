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

	PVs        []string   `json:"pvs,omitempty"`
	PVSelector PVSelector `json:"pvSelector,omitempty"`

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

	VGAttributes string   `json:"attributes,omitempty"`
	Tags         []string `json:"tags,omitempty"`

	ExtentSize  *resource.Quantity `json:"extentSize,omitempty"`
	ExtentCount int64              `json:"extentCount,omitempty"`

	SeqNo int64 `json:"seqno,omitempty"`

	Size *resource.Quantity `json:"size,omitempty"`
	Free *resource.Quantity `json:"free,omitempty"`

	PvCount        int64 `json:"pvCount,omitempty"`
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

// PVSelector represents the union of the results of one or more queries
// over a set of physical volume candidates; that is, it represents the OR of the selectors represented
// by the pv selector terms.
// +structType=atomic
type PVSelector struct {
	// Required. A list of node selector terms. The terms are ORed.
	// +listType=atomic
	PVSelectorTerms []PVSelectorTerm `json:"pvSelectorTerms" protobuf:"bytes,1,rep,name=pvSelectorTerms"`
}

// A null or empty pv selector term matches no objects. The requirements of
// them are ANDed.
// The TopologySelectorTerm type implements a subset of the PVSelectorTerm.
// +structType=atomic
type PVSelectorTerm struct {
	// A list of node selector requirements by node's labels.
	// +optional
	// +listType=atomic
	MatchLSBLK []LSBLKSelectorRequirement `json:"matchLSBLK,omitempty" protobuf:"bytes,1,rep,name=matchExpressions"`
}

// LSBLKSelectorRequirement is a selector that contains values, a key, and an operator
// that relates the key and values.
type LSBLKSelectorRequirement struct {
	// The label key that the selector applies to.
	Key LSBLKSelectorKey `json:"key" protobuf:"bytes,1,opt,name=key"`
	// Represents a key's relationship to a set of values.
	// Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
	Operator PVSelectorOperator `json:"operator" protobuf:"bytes,2,opt,name=operator,casttype=NodeSelectorOperator"`
	// An array of string values. If the operator is In or NotIn,
	// the values array must be non-empty. If the operator is Exists or DoesNotExist,
	// the values array must be empty. If the operator is Gt or Lt, the values
	// array must have a single element, which will be interpreted as a resource.Quantity.
	// This array is replaced during a strategic merge patch.
	// +optional
	// +listType=atomic
	Values []string `json:"values,omitempty" protobuf:"bytes,3,rep,name=values"`
}

// PVSelectorOperator is the set of operators that can be used in
// a node selector requirement.
// +enum
type PVSelectorOperator string

const (
	PVSelectorOpIn           PVSelectorOperator = "In"
	PVSelectorOpExists       PVSelectorOperator = "Exists"
	PVSelectorOpDoesNotExist PVSelectorOperator = "DoesNotExist"
	PVSelectorGt             PVSelectorOperator = "Gt"
)

// LSBLKSelectorKey is the type of key that can be used in a node selector requirement.
// +kubebuilder:validation:Enum=NAME;KNAME;PATH;"MAJ:MIN";FSAVAIL;FSSIZE;FSTYPE;FSUSED;"FSUSE%";FSROOTS;FSVER;MOUNTPOINT;MOUNTPOINTS;LABEL;UUID;PTUUID;PTTYPE;PARTTYPE;PARTTYPENAME;PARTLABEL;PARTUUID;PARTFLAGS;RA;RO;RM;HOTPLUG;MODEL;SERIAL;SIZE;STATE;OWNER;GROUP;MODE;ALIGNMENT;MIN-IO;OPT-IO;PHY-SEC;LOG-SEC;ROTA;SCHED;RQ-SIZE;TYPE;DISC-ALN;DISC-GRAN;DISC-MAX;DISC-ZERO;WSAME;WWN;RAND;PKNAME;HCTL;TRAN;SUBSYSTEMS;REV;VENDOR;ZONED;DAX
type LSBLKSelectorKey string

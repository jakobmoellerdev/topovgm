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

// VolumeGroupSpec defines the desired state of a VolumeGroup.
// It contains various fields that specify how the volume group should be configured and managed.
type VolumeGroupSpec struct {
	// NodeName is the name of the node where the volume group should be created.
	// This field is immutable because the volume group is not movable between nodes.
	// The NodeName is equivalent to the name of the Node itself.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="the node cannot be changed once set"
	NodeName string `json:"nodeName"`

	// NameOnNode is the name of the volume group on the node.
	// If not specified, the name is generated by the controller from the UID of the VolumeGroup resource.
	// If specified, the name must be unique among the volume groups on the node.
	// Additionally, the name must be acceptable for use as a volume group name by the lvm2 subsystem.
	// When changed, the volume group is renamed on the node.
	// However, the actual name on the Node may be different from the NameOnNode field value if this fails.
	// In this case, the actual Name is reported in VolumeGroupStatus.Name.
	NameOnNode *string `json:"nameOnNode,omitempty"`

	// PhysicalVolumeSelector is a selector for physical volumes that should be included in the volume group.
	// If empty, no physical volumes are included in the volume group.
	// If the selector fails to include at least one device, the VolumeGroup creation will fail.
	// This is done at runtime and after admission of the VolumeGroupSpec.
	PhysicalVolumeSelector PhysicalVolumeSelector `json:"physicalVolumeSelector"`

	// Tags is a list of tags to apply to the volume group.
	// Tags are used to group volume groups and to apply policies to them.
	// They can also be used on the host to apply policies to all volume groups with the same tag.
	// Tags are changeable after the volume group is created, and correspond to --addtag and --deltag operations.
	// Tags can only be controlled by a single field manager.
	// +listType=atomic
	Tags []string `json:"tags,omitempty"`

	// MaximumLogicalVolumes is the maximum number of logical volumes that can be created in the volume group.
	// This limit is enforced in lvm2 and is changeable after the volume group is created.
	// If set to 0 or omitted, there is no limit.
	// This field can be used to prevent the creation of too many logical volumes in the volume group and
	// should be set to a value that is appropriate for the use case if known in advance.
	MaximumLogicalVolumes *int64 `json:"maximumLogicalVolumes,omitempty"`

	// MaximumPhysicalVolumes is the maximum number of physical volumes that can be added to the volume group.
	// This limit is enforced in lvm2 and is changeable after the volume group is created.
	// If set to 0 or omitted, there is no limit.
	// This field can be used to prevent the addition of too many physical volumes to the volume group and
	// should be set to a value that is appropriate for the use case if known in advance.
	// It can also be used to discover faulty device selection if used with a generous PhysicalVolumeSelector.
	// In the case that the specified number of physical volumes is lower than the selected amount,
	// the VolumeGroup creation will fail.
	MaximumPhysicalVolumes *int64 `json:"maximumPhysicalVolumes,omitempty"`

	// PhysicalExtentSize is the physical extent size of pvs inside the volume group.
	// The value must be either a power of 2 of at least 1 sector (where the sector size is the
	// largest sector size of the PVs currently used in the VG), or at least 128Ki.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="the physical extent size cannot be changed once set, as it requires significant changes to the volume group. Once this value has been set, it is difficult to change without recreating the VG, unless no extents need moving. Before increasing the physical extent size, you might need to use lvresize, pvresize and/or pvmove so that everything fits. For example, every contiguous range of extents used in a LV must start and end on an extent boundary."
	PhysicalExtentSize *resource.Quantity `json:"physicalExtentSize,omitempty"`

	// MetadataSize is the approximate amount of space used for each VG metadata area. The size may be rounded.
	// If not set, the host default is used.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="the metadata size cannot be changed once set"
	MetadataSize *resource.Quantity `json:"metadataSize,omitempty"`

	// AllocationPolicy is the policy used to allocate extents in the volume group.
	// If not set, the host default is used.
	AllocationPolicy *AllocationPolicy `json:"allocationPolicy,omitempty"`

	// Align the start of a PV data area with a multiple of this number. To see the location of the first Physical Extent (PE) of an existing PV,
	// use pvs -o +pe_start. In addition, it may be shifted by an alignment offset, see DataAlignmentOffset.
	// Also specify an appropriate PhysicalExtentSize size when creating a VG.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="the data alignment cannot be changed once set"
	DataAlignment *resource.Quantity `json:"dataAlignment,omitempty"`

	// Shift the start of the PV data area by this additional offset.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="the data alignment offset cannot be changed once set"
	DataAlignmentOffset *resource.Quantity `json:"dataAlignmentOffset,omitempty"`

	// Restricts the devices that are visible and accessible to the command. Devices not listed will appear to be missing.
	// This overrides the devices file.
	// WARNING: older versions of lvm2 might not support this field
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="the devices cannot be changed once set"
	Devices []string `json:"devices,omitempty"`

	// DevicesFile is a file listing devices that LVM should use.
	// The file must exist in /etc/lvm/devices/ and is managed with the lvmdevices(8) command. This
	// overrides the lvm.conf(5) devices/devicesfile and devices/use_devicesfile settings.
	// WARNING: older versions of lvm2 might not support this field
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="the devices file cannot be changed once set"
	DevicesFile *string `json:"devicesFile,omitempty"`

	// Zero controls if the first 4 sectors (2048 bytes) of the device are wiped.
	// If not specified, the host default is used.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="zeroing cannot be changed once set"
	Zero *bool `json:"zero,omitempty"`

	// AutoActivation controls automatic activation on a VG or LV in that VG. Display the property with vgs or lvs "-o autoactivation".
	// When the autoactivation property is disabled, the VG or LV will not be activated by a command doing autoactivation (vgchange, lvchange, or pvscan using -aay.)
	// If autoactivation is disabled on a VG, no LVs will be autoactivated in that VG, and the LV autoactivation property has no effect.
	// If autoactivation is enabled on a VG, autoactivation can be disabled for individual LVs.
	// If not specified, the host default is used.
	AutoActivation *bool `json:"autoActivation,omitempty"`
}

// VolumeGroupStatus defines the observed state of VolumeGroup in lvm2.
type VolumeGroupStatus struct {
	// Name is the current name of the volume group on the node as visible in lvm2.
	// Corresponds to vg_name.
	Name string `json:"name"`

	// UUID is the UUID of the volume group.
	// Corresponds to vg_uuid.
	UUID string `json:"uuid,omitempty"`
	// SysID is the system ID of the VG indicating which host owns it.
	// Corresponds to vg_sysid or vg_systemid.
	SysID string `json:"sysid,omitempty"`

	// PVs is a list of physical volumes in the volume group.
	PVs []string `json:"pvs,omitempty"`

	// VGAttributes are various attributes of the volume group.
	// Corresponds to vg_attr.
	VGAttributes string `json:"attributes,omitempty"`
	// Tags are tags applied to the volume group.
	// Corresponds to vg_tags.
	Tags []string `json:"tags,omitempty"`

	// ExtentSize is the size of physical extents in the volume group.
	// Corresponds to vg_extent_size.
	ExtentSize *resource.Quantity `json:"extentSize,omitempty"`
	// ExtentCount is the total number of physical extents in the volume group.
	// Corresponds to vg_extent_count.
	ExtentCount int64 `json:"extentCount,omitempty"`

	// SeqNo is the revision number of internal metadata.
	// Corresponds to vg_seqno.
	SeqNo int64 `json:"seqno,omitempty"`

	// Size is the total size of the volume group.
	// Corresponds to vg_size.
	Size *resource.Quantity `json:"size,omitempty"`
	// Free is the total amount of free space in the volume group.
	// Corresponds to vg_free.
	Free *resource.Quantity `json:"free,omitempty"`

	// PvCount is the number of physical volumes in the volume group.
	// Corresponds to pv_count.
	PvCount int64 `json:"pvCount,omitempty"`
	// MissingPVCount is the number of physical volumes in the volume group which are missing.
	// Corresponds to vg_missing_pv_count.
	MissingPVCount int64 `json:"missingPvCount,omitempty"`
	// MaxPv is the maximum number of physical volumes allowed in the volume group.
	// Corresponds to max_pv.
	MaxPv int64 `json:"maxPv,omitempty"`

	// LvCount is the number of logical volumes in the volume group.
	// Corresponds to lv_count.
	LvCount int64 `json:"lvCount,omitempty"`
	// MaxLv is the maximum number of logical volumes allowed in the volume group.
	// Corresponds to max_lv.
	MaxLv int64 `json:"maxLv,omitempty"`

	// SnapCount is the number of snapshots in the volume group.
	// Corresponds to snap_count.
	SnapCount int64 `json:"snapCount,omitempty"`

	// MDACount is the number of metadata areas on the volume group.
	// Corresponds to vg_mda_count.
	MDACount int64 `json:"mdaCount,omitempty"`
	// MDAUsedCount is the number of metadata areas in use on the volume group.
	// Corresponds to vg_mda_used_count.
	MDAUsedCount int64 `json:"mdaUsedCount,omitempty"`

	// Conditions represent the latest available observations of an object's state.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VolumeGroup is the Schema for the volumegroups API.
// It represents a logical grouping of physical volumes (PVs) and logical volumes (LVs) managed by LVM2.
// This struct contains metadata about the volume group, its desired state (spec), and its observed state (status).
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

// PhysicalVolumeSelector represents the union of the results of one or more queries
// over a set of physical volume candidates; that is, it represents the OR of the selectors represented
// by the PVSelectorTerm(s).
// +listType=atomic
type PhysicalVolumeSelector []PVSelectorTerm

// PVSelectorTerm is a term that must be fullfilled by a physical volume candidate to be considered for the volume group.
// A null or empty pv selector term matches no objects.
// The requirements of them are ANDed.
// +structType=atomic
type PVSelectorTerm struct {
	// A list of node selector requirements by node's labels.
	// +optional
	// +listType=atomic
	MatchLSBLK []LSBLKSelectorRequirement `json:"matchLSBLK,omitempty"`
}

// LSBLKSelectorRequirement is a selector that contains values, a key, and an operator
// that relates the key and values.
type LSBLKSelectorRequirement struct {
	// The label key that the selector applies to.
	Key LSBLKSelectorKey `json:"key"`
	// Represents a key's relationship to a set of values.
	// Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.
	Operator PVSelectorOperator `json:"operator"`
	// An array of string values.
	// If the operator is PVSelectorOpIn, Values must be non-empty.
	// If the operator is PVSelectorOpExists or PVSelectorOpDoesNotExist, Values must be empty.
	// If the operator is PVSelectorGt, Values must have a single element, which will be interpreted as a resource.Quantity.
	// This array is replaced during a strategic merge patch.
	// +optional
	// +listType=atomic
	Values []string `json:"values,omitempty"`
}

// PVSelectorOperator is the set of operators that can be used in
// a node selector requirement.
// +enum
type PVSelectorOperator string

const (
	PVSelectorOpIn           PVSelectorOperator = "In"           // See PVSelectorOperator for more information.
	PVSelectorOpExists       PVSelectorOperator = "Exists"       // See PVSelectorOperator for more information.
	PVSelectorOpDoesNotExist PVSelectorOperator = "DoesNotExist" // See PVSelectorOperator for more information.
	PVSelectorGt             PVSelectorOperator = "Gt"           // See PVSelectorOperator for more information.
)

// LSBLKSelectorKey is the type of key that can be used in a node selector requirement.
// +kubebuilder:validation:Enum=NAME;KNAME;PATH;"MAJ:MIN";FSAVAIL;FSSIZE;FSTYPE;FSUSED;"FSUSE%";FSROOTS;FSVER;MOUNTPOINT;MOUNTPOINTS;LABEL;UUID;PTUUID;PTTYPE;PARTTYPE;PARTTYPENAME;PARTLABEL;PARTUUID;PARTFLAGS;RA;RO;RM;HOTPLUG;MODEL;SERIAL;SIZE;STATE;OWNER;GROUP;MODE;ALIGNMENT;MIN-IO;OPT-IO;PHY-SEC;LOG-SEC;ROTA;SCHED;RQ-SIZE;TYPE;DISC-ALN;DISC-GRAN;DISC-MAX;DISC-ZERO;WSAME;WWN;RAND;PKNAME;HCTL;TRAN;SUBSYSTEMS;REV;VENDOR;ZONED;DAX
type LSBLKSelectorKey string

// AllocationPolicy is the policy used to allocate extents in the volume group.
// Determines the allocation policy when a command needs to allocate Physical Extents (PEs) from the VG.
// Each VG and LV has an allocation policy which can be changed with vgchange/lvchange,
// or overridden on the command line. Normal applies common sense rules such as not placing parallel stripes on the same PV.
// Inherit applies the VG policy to an LV. Contiguous requires new PEs be placed adjacent to existing PEs.
// Cling places new PEs on the same PV as existing PEs in the same stripe of the LV.
// If there are sufficient PEs for an allocation, but Normal does not use them,
// Anywhere will use them even if it reduces performance, e.g. by placing two stripes on the same PV.
// Optional positional PV args can also be used to limit which PVs the command will use for allocation.
// See man lvm(8) for more information about allocation.
type AllocationPolicy string

const (
	Contiguous  AllocationPolicy = "contiguous"    // See AllocationPolicy for more information.
	Normal      AllocationPolicy = "normal"        // See AllocationPolicy for more information.
	Cling       AllocationPolicy = "cling"         // See AllocationPolicy for more information.
	ClingByTags AllocationPolicy = "cling_by_tags" // See AllocationPolicy for more information.
	Anywhere    AllocationPolicy = "anywhere"      // See AllocationPolicy for more information.
	Inherit     AllocationPolicy = "inherit"       // See AllocationPolicy for more information.
)

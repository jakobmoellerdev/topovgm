//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LSBLKSelectorRequirement) DeepCopyInto(out *LSBLKSelectorRequirement) {
	*out = *in
	if in.Values != nil {
		in, out := &in.Values, &out.Values
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LSBLKSelectorRequirement.
func (in *LSBLKSelectorRequirement) DeepCopy() *LSBLKSelectorRequirement {
	if in == nil {
		return nil
	}
	out := new(LSBLKSelectorRequirement)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PVSelector) DeepCopyInto(out *PVSelector) {
	*out = *in
	if in.PVSelectorTerms != nil {
		in, out := &in.PVSelectorTerms, &out.PVSelectorTerms
		*out = make([]PVSelectorTerm, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PVSelector.
func (in *PVSelector) DeepCopy() *PVSelector {
	if in == nil {
		return nil
	}
	out := new(PVSelector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PVSelectorTerm) DeepCopyInto(out *PVSelectorTerm) {
	*out = *in
	if in.MatchLSBLK != nil {
		in, out := &in.MatchLSBLK, &out.MatchLSBLK
		*out = make([]LSBLKSelectorRequirement, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PVSelectorTerm.
func (in *PVSelectorTerm) DeepCopy() *PVSelectorTerm {
	if in == nil {
		return nil
	}
	out := new(PVSelectorTerm)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeGroup) DeepCopyInto(out *VolumeGroup) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeGroup.
func (in *VolumeGroup) DeepCopy() *VolumeGroup {
	if in == nil {
		return nil
	}
	out := new(VolumeGroup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VolumeGroup) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeGroupList) DeepCopyInto(out *VolumeGroupList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VolumeGroup, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeGroupList.
func (in *VolumeGroupList) DeepCopy() *VolumeGroupList {
	if in == nil {
		return nil
	}
	out := new(VolumeGroupList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *VolumeGroupList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeGroupSpec) DeepCopyInto(out *VolumeGroupSpec) {
	*out = *in
	if in.NameOnNode != nil {
		in, out := &in.NameOnNode, &out.NameOnNode
		*out = new(string)
		**out = **in
	}
	if in.PVs != nil {
		in, out := &in.PVs, &out.PVs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.PVSelector.DeepCopyInto(&out.PVSelector)
	if in.Devices != nil {
		in, out := &in.Devices, &out.Devices
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.DevicesFile != nil {
		in, out := &in.DevicesFile, &out.DevicesFile
		*out = new(string)
		**out = **in
	}
	if in.Tags != nil {
		in, out := &in.Tags, &out.Tags
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.MaximumLogicalVolumes != nil {
		in, out := &in.MaximumLogicalVolumes, &out.MaximumLogicalVolumes
		*out = new(int)
		**out = **in
	}
	if in.MaximumPhysicalVolumes != nil {
		in, out := &in.MaximumPhysicalVolumes, &out.MaximumPhysicalVolumes
		*out = new(int)
		**out = **in
	}
	if in.PhysicalExtentSize != nil {
		in, out := &in.PhysicalExtentSize, &out.PhysicalExtentSize
		x := (*in).DeepCopy()
		*out = &x
	}
	if in.AllocationPolicy != nil {
		in, out := &in.AllocationPolicy, &out.AllocationPolicy
		*out = new(string)
		**out = **in
	}
	if in.DataAlignment != nil {
		in, out := &in.DataAlignment, &out.DataAlignment
		x := (*in).DeepCopy()
		*out = &x
	}
	if in.DataAlignmentOffset != nil {
		in, out := &in.DataAlignmentOffset, &out.DataAlignmentOffset
		x := (*in).DeepCopy()
		*out = &x
	}
	if in.Zero != nil {
		in, out := &in.Zero, &out.Zero
		*out = new(bool)
		**out = **in
	}
	if in.AutoActivation != nil {
		in, out := &in.AutoActivation, &out.AutoActivation
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeGroupSpec.
func (in *VolumeGroupSpec) DeepCopy() *VolumeGroupSpec {
	if in == nil {
		return nil
	}
	out := new(VolumeGroupSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeGroupStatus) DeepCopyInto(out *VolumeGroupStatus) {
	*out = *in
	if in.PVs != nil {
		in, out := &in.PVs, &out.PVs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Tags != nil {
		in, out := &in.Tags, &out.Tags
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.ExtentSize != nil {
		in, out := &in.ExtentSize, &out.ExtentSize
		x := (*in).DeepCopy()
		*out = &x
	}
	if in.Size != nil {
		in, out := &in.Size, &out.Size
		x := (*in).DeepCopy()
		*out = &x
	}
	if in.Free != nil {
		in, out := &in.Free, &out.Free
		x := (*in).DeepCopy()
		*out = &x
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeGroupStatus.
func (in *VolumeGroupStatus) DeepCopy() *VolumeGroupStatus {
	if in == nil {
		return nil
	}
	out := new(VolumeGroupStatus)
	in.DeepCopyInto(out)
	return out
}

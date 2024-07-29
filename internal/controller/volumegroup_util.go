package controller

import (
	"fmt"

	"github.com/jakobmoellerdev/lvm2go"
	"github.com/topolvm/topovgm/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func nameOnNode(vg *v1alpha1.VolumeGroup) lvm2go.VolumeGroupName {
	name := lvm2go.VolumeGroupName(vg.GetName())
	if vg.Spec.NameOnNode != nil {
		name = lvm2go.VolumeGroupName(*vg.Spec.NameOnNode)
	}
	return name
}

func setStatusFromLVMVolumeGroup(vg *v1alpha1.VolumeGroup, lvmvg *lvm2go.VolumeGroup) (err error) {
	vg.Status.Name = string(lvmvg.Name)
	vg.Status.UUID = lvmvg.UUID
	vg.Status.SysID = lvmvg.SysID
	vg.Status.VGAttributes = lvmvg.VGAttributes
	vg.Status.Tags = lvmvg.Tags
	vg.Status.ExtentSize, err = convertSizeToQuantity(lvmvg.ExtentSize)
	if err != nil {
		return err
	}
	vg.Status.ExtentCount = lvmvg.ExtentCount
	vg.Status.SeqNo = lvmvg.SeqNo
	vg.Status.Size, err = convertSizeToQuantity(lvmvg.Size)
	if err != nil {
		return err
	}
	vg.Status.Free, err = convertSizeToQuantity(lvmvg.Free)
	if err != nil {
		return err
	}
	vg.Status.PvCount = lvmvg.PvCount
	vg.Status.MissingPVCount = lvmvg.MissingPVCount
	vg.Status.MaxPv = lvmvg.MaxPv
	vg.Status.LvCount = lvmvg.LvCount
	vg.Status.MaxLv = lvmvg.MaxLv
	vg.Status.SnapCount = lvmvg.SnapCount
	vg.Status.MDACount = lvmvg.MDACount
	vg.Status.MDAUsedCount = lvmvg.MDAUsedCount
	vg.Status.MDACopies = lvmvg.MDACopies
	return nil
}

func convertToVGCreateOptions(vg *v1alpha1.VolumeGroup) (*lvm2go.VGCreateOptions, error) {
	opts := &lvm2go.VGCreateOptions{
		VolumeGroupName: nameOnNode(vg),
	}

	if vg.Spec.Tags != nil {
		opts.Tags = vg.Spec.Tags
	}

	if vg.Spec.PVs != nil {
		opts.PhysicalVolumeNames = convertToPhysicalVolumeNames(vg.Spec.PVs)
	}

	if vg.Spec.AutoActivation != nil {
		opts.AutoActivation = convertToAutoActivation(vg.Spec.AutoActivation)
	}

	if vg.Spec.Zero != nil {
		opts.Zero = convertToZero(vg.Spec.Zero)
	}

	if vg.Spec.AllocationPolicy != nil {
		opts.AllocationPolicy = lvm2go.AllocationPolicy(*vg.Spec.AllocationPolicy)
	}

	if vg.Spec.PhysicalExtentSize != nil {
		physicalExtentSize, err := convertQuantityToSize(vg.Spec.PhysicalExtentSize)
		if err != nil {
			return nil, err
		}
		opts.PhysicalExtentSize = lvm2go.PhysicalExtentSize(physicalExtentSize)
	}

	if vg.Spec.Devices != nil {
		opts.Devices = vg.Spec.Devices
	}

	if vg.Spec.DevicesFile != nil {
		opts.DevicesFile = lvm2go.DevicesFile(*vg.Spec.DevicesFile)
	}

	return opts, nil
}

func convertToPhysicalVolumeNames(pvs []string) []lvm2go.PhysicalVolumeName {
	physicalVolumes := make([]lvm2go.PhysicalVolumeName, 0, len(pvs))
	for _, pv := range pvs {
		physicalVolumes = append(physicalVolumes, lvm2go.PhysicalVolumeName(pv))
	}
	return physicalVolumes
}

func convertToAutoActivation(autoActivation *bool) lvm2go.AutoActivation {
	if autoActivation == nil {
		return lvm2go.SetAutoActivate
	}
	return lvm2go.SetNoAutoActivate
}

func convertToZero(zero *bool) lvm2go.Zero {
	if zero == nil || *zero {
		return lvm2go.DoNotZeroVolume
	}
	return lvm2go.ZeroVolume
}

func convertQuantityToSize(q *resource.Quantity) (lvm2go.Size, error) {
	if q == nil {
		return lvm2go.InvalidSize, fmt.Errorf("quantity is nil")
	}
	size := lvm2go.NewSize(q.AsApproximateFloat64(), lvm2go.UnitBytes)
	if err := size.Validate(); err != nil {
		return lvm2go.InvalidSize, err
	}
	return size, nil
}

func convertSizeToQuantity(size lvm2go.Size) (*resource.Quantity, error) {
	q, err := size.ToUnit(lvm2go.UnitBytes)
	if err != nil {
		return nil, err
	}
	return resource.NewQuantity(int64(q.Val), resource.DecimalSI), nil
}

func setSyncedOnHostCondition(conditions *[]metav1.Condition, err error) {
	condition := metav1.Condition{
		Type: ConditionTypeVolumeGroupSyncedOnNode,
	}

	if err == nil {
		condition.Status = metav1.ConditionTrue
		condition.Reason = ReasonVolumeGroupCreated
		condition.Message = MessageVolumeGroupCreated
	} else {
		condition.Status = metav1.ConditionFalse
		condition.Reason = ReasonVolumeGroupCreationFailed
		condition.Message = err.Error()
	}

	if conditions == nil {
		*conditions = []metav1.Condition{}
	}

	meta.SetStatusCondition(conditions, condition)
}

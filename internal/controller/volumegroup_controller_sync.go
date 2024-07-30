package controller

import (
	"context"
	"errors"
	"fmt"

	"github.com/jakobmoellerdev/lvm2go"
	"github.com/topolvm/topovgm/api/v1alpha1"
	"github.com/topolvm/topovgm/internal/utils"
)

// sync synchronizes the desired state of the volume group with the actual state from lvm2.
func (r *VolumeGroupReconciler) sync(
	ctx context.Context,
	vg *v1alpha1.VolumeGroup,
	lvmvg *lvm2go.VolumeGroup,
) error {
	SetSyncedOnHostDefault(&vg.Status.Conditions, vg.GetGeneration())

	syncers := []func(context.Context, *v1alpha1.VolumeGroup, *lvm2go.VolumeGroup) error{
		r.syncTags,
		r.syncPVs,
	}

	errs := make([]error, 0, len(syncers))
	for _, sync := range syncers {
		errs = append(errs, sync(ctx, vg, lvmvg))
	}
	err := errors.Join(errs...)

	if err != nil {
		SetSyncedOnHostCreationFailed(&vg.Status.Conditions, vg.GetGeneration(), err)
	} else {
		SetSyncedOnHostCreationOK(&vg.Status.Conditions, vg.GetGeneration())
	}

	return err
}

// syncTags calculates the difference between the desired tags and the actual tags and applies the difference to the volume group.
func (r *VolumeGroupReconciler) syncTags(
	ctx context.Context,
	vg *v1alpha1.VolumeGroup,
	lvm *lvm2go.VolumeGroup,
) error {
	name := nameOnNode(vg)

	return utils.SequentialTwoWaySync(
		vg.Spec.Tags,
		lvm.Tags,
		func(tags []string) error {
			return r.LVM.VGChange(ctx, name, lvm2go.Tags(tags))
		},
		func(tags []string) error {
			return r.LVM.VGChange(ctx, name, lvm2go.DelTags(tags))
		},
	)
}

// syncPVs calculates the difference between the desired PVs and the actual PVs and applies the difference to the volume group.
func (r *VolumeGroupReconciler) syncPVs(
	ctx context.Context,
	vg *v1alpha1.VolumeGroup,
	lvm *lvm2go.VolumeGroup,
) error {
	name := nameOnNode(vg)

	pvs, err := r.LVM.PVs(ctx, lvm.Name)
	if err != nil {
		return fmt.Errorf("could not get pvs for calculation of state diff: %w", err)
	}

	return utils.SequentialTwoWaySync(
		utils.Map(vg.Spec.PVs, func(pv string) lvm2go.PhysicalVolumeName {
			return lvm2go.PhysicalVolumeName(pv)
		}),
		utils.Map(pvs, func(pv *lvm2go.PhysicalVolume) lvm2go.PhysicalVolumeName {
			return pv.Name
		}),
		func(names []lvm2go.PhysicalVolumeName) error {
			return r.LVM.VGExtend(ctx, name, lvm2go.PhysicalVolumeNames(names))
		},
		func(names []lvm2go.PhysicalVolumeName) error {
			return r.LVM.VGReduce(ctx, name, lvm2go.PhysicalVolumeNames(names))
		},
	)
}

// syncStatus synchronizes the status of the volume group with the actual state from lvm2.
func (r *VolumeGroupReconciler) syncStatus(
	ctx context.Context,
	vg *v1alpha1.VolumeGroup,
	lvm *lvm2go.VolumeGroup,
) (err error) {
	status := &vg.Status
	pvs, err := r.LVM.PVs(ctx, lvm.Name)
	if err != nil {
		return fmt.Errorf("could not get pvs for status summary: %w", err)
	}

	status.PVs = utils.Map(pvs, func(pv *lvm2go.PhysicalVolume) string {
		return string(pv.Name)
	})

	status.Name = string(lvm.Name)
	status.UUID = lvm.UUID
	status.SysID = lvm.SysID
	status.VGAttributes = lvm.VGAttributes
	status.Tags = lvm.Tags
	status.ExtentSize, err = convertSizeToQuantity(lvm.ExtentSize)
	if err != nil {
		return err
	}
	status.ExtentCount = lvm.ExtentCount
	status.SeqNo = lvm.SeqNo
	status.Size, err = convertSizeToQuantity(lvm.Size)
	if err != nil {
		return err
	}
	status.Free, err = convertSizeToQuantity(lvm.Free)
	if err != nil {
		return err
	}
	status.PvCount = lvm.PvCount
	status.MissingPVCount = lvm.MissingPVCount
	status.MaxPv = lvm.MaxPv
	status.LvCount = lvm.LvCount
	status.MaxLv = lvm.MaxLv
	status.SnapCount = lvm.SnapCount
	status.MDACount = lvm.MDACount
	status.MDAUsedCount = lvm.MDAUsedCount
	status.MDACopies = lvm.MDACopies
	return nil
}

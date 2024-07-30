package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jakobmoellerdev/lvm2go"
	"github.com/topolvm/topovgm/api/v1alpha1"
	"github.com/topolvm/topovgm/internal/utils"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

	logger := log.FromContext(ctx).WithValues("vg", vg.Name)

	start := time.Now()
	logger.V(1).Info("syncing volume group")
	defer func() {
		logger.V(1).Info("finished syncing volume group", "duration", time.Since(start))
	}()

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

	desiredState, err := pvsFromSpec(ctx, vg)
	if err != nil {
		return fmt.Errorf("could not get physical volume names to sync spec: %w", err)
	}

	pvs, err := r.LVM.PVs(ctx, lvm.Name, lvm2go.UnitBytes)
	if err != nil {
		return fmt.Errorf("could not get pvs for calculation of state diff: %w", err)
	}
	currentState := utils.Map(pvs, func(pv *lvm2go.PhysicalVolume) lvm2go.PhysicalVolumeName {
		return pv.Name
	})

	return utils.SequentialTwoWaySync(
		desiredState,
		currentState,
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
	pvs, err := r.LVM.PVs(ctx, lvm.Name)
	if err != nil {
		return fmt.Errorf("could not get pvs for status summary: %w", err)
	}

	vg.Status.PVs = utils.Map(pvs, func(pv *lvm2go.PhysicalVolume) string {
		return string(pv.Name)
	})

	vg.Status.Name = string(lvm.Name)
	vg.Status.UUID = lvm.UUID
	vg.Status.SysID = lvm.SysID
	vg.Status.VGAttributes = lvm.VGAttributes
	vg.Status.Tags = lvm.Tags
	vg.Status.ExtentSize, err = convertSizeToQuantity(lvm.ExtentSize)
	if err != nil {
		return err
	}
	vg.Status.ExtentCount = lvm.ExtentCount
	vg.Status.SeqNo = lvm.SeqNo
	vg.Status.Size, err = convertSizeToQuantity(lvm.Size)
	if err != nil {
		return err
	}
	vg.Status.Free, err = convertSizeToQuantity(lvm.Free)
	if err != nil {
		return err
	}
	vg.Status.PvCount = lvm.PvCount
	vg.Status.MissingPVCount = lvm.MissingPVCount
	vg.Status.MaxPv = lvm.MaxPv
	vg.Status.LvCount = lvm.LvCount
	vg.Status.MaxLv = lvm.MaxLv
	vg.Status.SnapCount = lvm.SnapCount
	vg.Status.MDACount = lvm.MDACount
	vg.Status.MDAUsedCount = lvm.MDAUsedCount
	vg.Status.MDACopies = lvm.MDACopies
	return nil
}

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
		r.syncMaximumVolumes,
		r.syncAllocationPolicy,
		r.syncAutoActivation,
		r.syncName,
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

	// Handle device loss based on the DeviceLossSynchronizationPolicy.
	// If activated and the volume group is missing physical volumes, remove them.
	// If set to Fail and the volume group is missing physical volumes, return an error.
	if lvm2go.IsLVMErrVGMissingPVs(err) {
		if missingVG, missingPV, lastWritePath, ok := lvm2go.LVMErrVGMissingPVsDetails(err); ok {
			logger = logger.WithValues(
				"missingVolumeGroup", missingVG,
				"missingPhysicalVolume", missingPV,
				"lastWritePath", lastWritePath,
			)
		}

		if vg.Spec.DeviceLossSynchronizationPolicy != v1alpha1.DeviceLossSynchronizationPolicyFail {
			logger.Info("device loss detected, removing missing physical volumes")
			opts := []lvm2go.Argument{lvm2go.RemoveMissing(true)}
			if vg.Spec.DeviceLossSynchronizationPolicy == v1alpha1.DeviceLossSynchronizationPolicyForceRemoveMissing {
				opts = append(opts, lvm2go.Force(true))
			}
			if err := r.LVM.VGReduce(ctx, lvmvg.Name, lvm2go.RemoveMissing(true)); err != nil {
				return fmt.Errorf("could not remove missing physical volumes (attempted due to DeviceLossSynchronizationPolicy): %w", err)
			}
			return r.sync(ctx, vg, lvmvg)
		}

		logger.Info("device loss detected")
		SetSyncedOnHostCreationFailed(&vg.Status.Conditions, vg.GetGeneration(), err)
		return err
	}

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
	name := getNameOnNode(vg)

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
	name := getNameOnNode(vg)

	desiredState, err := getPhysicalVolumeNames(ctx, vg)
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
			args := []lvm2go.VGReduceOption{name, lvm2go.PhysicalVolumeNames(names)}
			switch vg.Spec.DeviceRemovalVolumePolicy {
			case v1alpha1.DeviceRemovalVolumePolicyMoveAndReduce:
				for _, pv := range names {
					if err := r.LVM.PVMove(ctx, pv, lvm2go.PhysicalVolumeNames(desiredState)); err != nil {
						return err
					}
				}
			case v1alpha1.DeviceRemovalVolumePolicyForceReduce:
				args = append(args, lvm2go.Force(true))
			}
			return r.LVM.VGReduce(ctx, args...)
		},
	)
}

// syncName calculates the difference between the desired name and the actual name and renames the volume group if necessary.
func (r *VolumeGroupReconciler) syncName(
	ctx context.Context,
	vg *v1alpha1.VolumeGroup,
	lvm *lvm2go.VolumeGroup,
) error {
	desired := getNameOnNode(vg)
	if lvm.Name == desired {
		return nil
	}
	return r.LVM.VGRename(ctx, lvm.Name, desired)
}

// syncMaximumVolumes synchronizes the maximum number of physical and logical volumes in the volume group.
func (r *VolumeGroupReconciler) syncMaximumVolumes(
	ctx context.Context,
	vg *v1alpha1.VolumeGroup,
	lvm *lvm2go.VolumeGroup,
) error {
	if vg.Spec.MaximumPhysicalVolumes != nil && lvm.MaxPv != *vg.Spec.MaximumPhysicalVolumes {
		if err := r.LVM.VGChange(ctx, lvm.Name, lvm2go.MaximumPhysicalVolumes(*vg.Spec.MaximumPhysicalVolumes)); err != nil {
			return fmt.Errorf("could not set maximum physical volumes: %w", err)
		}
	}
	if vg.Spec.MaximumLogicalVolumes != nil && lvm.MaxLv != *vg.Spec.MaximumLogicalVolumes {
		if err := r.LVM.VGChange(ctx, lvm.Name, lvm2go.MaximumLogicalVolumes(*vg.Spec.MaximumLogicalVolumes)); err != nil {
			return fmt.Errorf("could not set maximum logical volumes: %w", err)
		}
	}

	return nil
}

// syncAllocationPolicy synchronizes the allocation policy of the volume group.
func (r *VolumeGroupReconciler) syncAllocationPolicy(
	ctx context.Context,
	vg *v1alpha1.VolumeGroup,
	lvm *lvm2go.VolumeGroup,
) error {
	if vg.Spec.AllocationPolicy == nil {
		return nil
	}

	desired := lvm2go.AllocationPolicy(utils.ToSnakeCase(string(*vg.Spec.AllocationPolicy)))

	if lvm.AllocationPolicy == desired {
		return nil
	}

	return r.LVM.VGChange(ctx, lvm.Name, desired)
}

func (r *VolumeGroupReconciler) syncAutoActivation(
	ctx context.Context,
	vg *v1alpha1.VolumeGroup,
	lvm *lvm2go.VolumeGroup,
) error {
	if vg.Spec.AutoActivation == nil {
		return nil
	}

	desired := convertToAutoActivation(vg.Spec.AutoActivation)

	if lvm.AutoActivation.True() && desired == lvm2go.SetAutoActivate {
		return nil
	} else if !lvm.AutoActivation.True() && desired == lvm2go.SetNoAutoActivate {
		return nil
	}

	return r.LVM.VGChange(ctx, lvm.Name, desired)
}

// syncStatus synchronizes the status of the volume group with the actual state from lvm2.
func (r *VolumeGroupReconciler) syncStatus(
	ctx context.Context,
	vg *v1alpha1.VolumeGroup,
	lvm *lvm2go.VolumeGroup,
) (err error) {
	pvs, err := r.LVM.PVs(ctx, lvm.Name, lvm2go.UnitBytes)
	if err != nil {
		return fmt.Errorf("could not get pvs for status summary: %w", err)
	}

	vg.Status.PhysicalVolumes = make([]v1alpha1.PhysicalVolumeStatus, len(pvs))
	for i, pv := range pvs {
		vg.Status.PhysicalVolumes[i].Name = string(pv.Name)
		vg.Status.PhysicalVolumes[i].UUID = pv.UUID
		if vg.Status.PhysicalVolumes[i].DeviceSize, err = convertSizeToQuantity(pv.DevSize); err != nil {
			return err
		}
		if vg.Status.PhysicalVolumes[i].Size, err = convertSizeToQuantity(pv.Size); err != nil {
			return err
		}
		if vg.Status.PhysicalVolumes[i].Free, err = convertSizeToQuantity(pv.Free); err != nil {
			return err
		}
		if vg.Status.PhysicalVolumes[i].Used, err = convertSizeToQuantity(pv.Used); err != nil {
			return err
		}
		if vg.Status.PhysicalVolumes[i].MetadataAreaFree, err = convertSizeToQuantity(pv.MdaFree); err != nil {
			return err
		}
		if vg.Status.PhysicalVolumes[i].MetadataAreaSize, err = convertSizeToQuantity(pv.MdaSize); err != nil {
			return err
		}
		if vg.Status.PhysicalVolumes[i].PhysicalExtentStart, err = convertSizeToQuantity(pv.PeStart); err != nil {
			return err
		}
		vg.Status.PhysicalVolumes[i].MetadataAreaCount = pv.MdaCount
		vg.Status.PhysicalVolumes[i].MetadataAreaUsedCount = pv.MdaUsedCount
		vg.Status.PhysicalVolumes[i].Tags = pv.Tags
		vg.Status.PhysicalVolumes[i].DeviceID = pv.DeviceID
		vg.Status.PhysicalVolumes[i].DeviceIDType = pv.DeviceIDType
		vg.Status.PhysicalVolumes[i].Attributes = pv.Attr.String()
		vg.Status.PhysicalVolumes[i].Minor = pv.Minor
		vg.Status.PhysicalVolumes[i].Major = pv.Major
	}

	if vg.Status.ExtentSize, err = convertSizeToQuantity(lvm.ExtentSize); err != nil {
		return err
	}
	if vg.Status.Size, err = convertSizeToQuantity(lvm.Size); err != nil {
		return err
	}
	if vg.Status.Free, err = convertSizeToQuantity(lvm.Free); err != nil {
		return err
	}
	vg.Status.Name = string(lvm.Name)
	vg.Status.UUID = lvm.UUID
	vg.Status.SysID = lvm.SysID
	vg.Status.Attributes = lvm.Attr.String()
	vg.Status.Tags = lvm.Tags
	vg.Status.ExtentCount = lvm.ExtentCount
	vg.Status.SequenceNumber = lvm.SeqNo

	vg.Status.PhysicalVolumeCount = lvm.PvCount
	vg.Status.MissingPhysicalVolumeCount = lvm.MissingPVCount
	vg.Status.MaximumPhysicalVolumes = lvm.MaxPv
	vg.Status.LogicalVolumeCount = lvm.LvCount
	vg.Status.MaximumLogicalVolumes = lvm.MaxLv
	vg.Status.SnapshotCount = lvm.SnapCount
	vg.Status.MetadataAreaCount = lvm.MDACount
	vg.Status.MetadataAreaUsedCount = lvm.MDAUsedCount
	return nil
}

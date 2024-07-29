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

package controller

import (
	"context"
	"errors"
	"fmt"

	"github.com/jakobmoellerdev/lvm2go"
	"github.com/topolvm/topovgm/internal/utils"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/topolvm/topovgm/api/v1alpha1"
)

const (
	// ConditionTypeVolumeGroupSyncedOnNode is a condition type that indicates whether the volume group is present on the host node.
	ConditionTypeVolumeGroupSyncedOnNode = "VolumeGroupSyncedOnNode"
	ReasonVolumeGroupCreated             = "VolumeGroupCreated"
	ReasonVolumeGroupCreationFailed      = "VolumeGroupCreationFailed"
	MessageVolumeGroupCreated            = "The volume group is present on the node and discoverable in the lvm2 subsystem."
)

// VolumeGroupReconciler reconciles a VolumeGroup object
type VolumeGroupReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	LVM      lvm2go.Client
	NodeName string
}

// SetupWithManager sets up the controller with the Manager.
func (r *VolumeGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.VolumeGroup{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=topolvm.io,resources=volumegroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=topolvm.io,resources=volumegroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=topolvm.io,resources=volumegroups/finalizers,verbs=update

// Reconcile reconciles a v1alpha1.VolumeGroup object
func (r *VolumeGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.V(1).Info("reconciling")

	vg := &v1alpha1.VolumeGroup{}
	if err := r.Client.Get(ctx, req.NamespacedName, vg); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if r.NodeName != vg.Spec.NodeName {
		logger.V(1).Info("skipping VolumeGroup due to mismatched .spec.nodeName",
			"expected", vg.Spec.NodeName,
			"actual", r.NodeName,
		)
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, r.reconcile(ctx, vg)
}

func (r *VolumeGroupReconciler) reconcile(ctx context.Context, vg *v1alpha1.VolumeGroup) error {
	name := nameOnNode(vg)

	lvmvg, err := r.LVM.VG(ctx, name)
	if errors.Is(err, lvm2go.ErrVolumeGroupNotFound) {
		if !vg.GetDeletionTimestamp().IsZero() {
			return nil
		}

		if opts, err := convertToVGCreateOptions(vg); err != nil {
			return fmt.Errorf("failed to convert to VGCreateOptions: %w", err)
		} else if err := r.LVM.VGCreate(ctx, opts); err != nil {
			return r.updateStatus(ctx, vg, err)
		}
		lvmvg, err = r.LVM.VG(ctx, name)
	} else if !vg.GetDeletionTimestamp().IsZero() {
		if err := r.LVM.VGRemove(ctx, name); err != nil {
			return fmt.Errorf("failed to remove volume group: %w", err)
		}
		return nil
	}

	if err != nil {
		return r.updateStatus(ctx, vg, err)
	}

	if err := r.sync(ctx, vg, lvmvg); err != nil {
		return fmt.Errorf("failed to sync: %w", err)
	}

	return r.updateStatusWithVG(ctx, vg, lvmvg, err)
}

func (r *VolumeGroupReconciler) updateStatusWithVG(
	ctx context.Context,
	vg *v1alpha1.VolumeGroup,
	lvmvg *lvm2go.VolumeGroup,
	err error,
) error {
	if err := setStatusFromLVMVolumeGroup(vg, lvmvg); err != nil {
		return fmt.Errorf("failed to set status from LVM volume group: %w", err)
	}
	return r.updateStatus(ctx, vg, err)
}

func (r *VolumeGroupReconciler) updateStatus(
	ctx context.Context,
	vg *v1alpha1.VolumeGroup,
	err error,
) error {
	setSyncedOnHostCondition(&vg.Status.Conditions, err)
	if err := r.Client.Status().Update(ctx, vg); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	return err
}

func (r *VolumeGroupReconciler) sync(
	ctx context.Context,
	vg *v1alpha1.VolumeGroup,
	lvmvg *lvm2go.VolumeGroup,
) error {
	if newFromSpec := utils.InLeftButNotInRight(vg.Spec.Tags, lvmvg.Tags); len(newFromSpec) > 0 {
		if err := r.LVM.VGChange(ctx, lvmvg.Name, lvm2go.Tags(newFromSpec)); err != nil {
			return fmt.Errorf("failed to synchronize tags: %w", err)
		}
	}
	if oldFromLVM := utils.InLeftButNotInRight(lvmvg.Tags, vg.Spec.Tags); len(oldFromLVM) > 0 {
		if err := r.LVM.VGChange(ctx, lvmvg.Name, lvm2go.DelTags(oldFromLVM)); err != nil {
			return fmt.Errorf("failed to synchronize tags: %w", err)
		}
	}

	pvs, err := r.LVM.PVs(ctx, lvmvg.Name)
	if err != nil {
		return fmt.Errorf("failed to synchronize pvs: %w", err)
	}
	inSpec := utils.ConvertSlice(vg.Spec.PVs, func(pv string) lvm2go.PhysicalVolumeName {
		return lvm2go.PhysicalVolumeName(pv)
	})
	inLVM := utils.ConvertSlice(pvs, func(pv *lvm2go.PhysicalVolume) lvm2go.PhysicalVolumeName {
		return pv.Name
	})
	if newFromSpec := utils.InLeftButNotInRight(inSpec, inLVM); len(newFromSpec) > 0 {
		if err := r.LVM.VGExtend(ctx, lvmvg.Name, lvm2go.PhysicalVolumeNames(newFromSpec)); err != nil {
			return fmt.Errorf("failed to extend volume group: %w", err)
		}
	}
	if oldFromLVM := utils.InLeftButNotInRight(inLVM, inSpec); len(oldFromLVM) > 0 {
		if err := r.LVM.VGReduce(ctx, lvmvg.Name, lvm2go.PhysicalVolumeNames(oldFromLVM)); err != nil {
			return fmt.Errorf("failed to reduce volume group: %w", err)
		}
	}

	return nil
}

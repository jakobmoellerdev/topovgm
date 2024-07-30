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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/topolvm/topovgm/api/v1alpha1"
)

const (
	VolumeGroupFinalizer = "topolvm.io/volumegroup-removal-on-node"
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

	if !vg.GetDeletionTimestamp().IsZero() {
		if err := r.LVM.VGRemove(ctx, name); err != nil && !lvm2go.IsLVMNotFound(err) {
			return fmt.Errorf("failed to remove volume group: %w", err)
		}
		if updated := controllerutil.RemoveFinalizer(vg, VolumeGroupFinalizer); updated {
			return r.Update(ctx, vg)
		}
		return nil
	}

	lvm, err := r.LVM.VG(ctx, name)

	if errors.Is(err, lvm2go.ErrVolumeGroupNotFound) {
		err = r.initializeVG(ctx, vg)
	}

	if err != nil {
		return r.Client.Status().Update(ctx, vg)
	}

	if updated := controllerutil.AddFinalizer(vg, VolumeGroupFinalizer); updated {
		return r.Update(ctx, vg)
	}

	if err = r.sync(ctx, vg, lvm); err != nil {
		err = fmt.Errorf("failed to sync volume group with lvm2: %w", err)
	}

	if statusErr := r.syncStatus(ctx, vg, lvm); statusErr != nil {
		err = errors.Join(err, fmt.Errorf("failed to sync status from lvm2 into volume group: %w", statusErr))
	}

	return errors.Join(err, r.Client.Status().Update(ctx, vg))
}

func (r *VolumeGroupReconciler) initializeVG(ctx context.Context, vg *v1alpha1.VolumeGroup) error {
	opts, err := convertToVGCreateOptions(vg)
	if err != nil {
		return fmt.Errorf("failed to convert VolumeGroup to VGCreateOptions: %w", err)
	}

	if err = r.LVM.VGCreate(ctx, opts); err != nil {
		SetSyncedOnHostCreationFailed(&vg.Status.Conditions, vg.GetGeneration(), err)
	}

	return err
}

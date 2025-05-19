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
	"os/exec" // Kept for exec.ExitError, just in case
	"strings" // Added for strings.Contains
	"time"

	"github.com/azalio/lvm2go"
	"github.com/azalio/topovgm/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	// "github.com/topolvm/topovgm/api/v1alpha1"
)

const (
	VolumeGroupFinalizer = "topolvm.io/volumegroup-removal-on-node"
)

// VolumeGroupReconciler reconciles a VolumeGroup object
type VolumeGroupReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	LVM          lvm2go.Client
	NodeName     string
	SyncInterval time.Duration
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

	name := getNameOnNode(vg)
	logger = logger.WithValues("node", r.NodeName, "lvm_name", name)

	if !vg.GetDeletionTimestamp().IsZero() {
		logger.V(1).Info("removing volume group from host")
		// If the VG is being deleted, we need to remove it from the host.
		// If the deletion grace period is set and the deletion timestamp has passed, we need to force the removal.
		forceAt := vg.GetDeletionTimestamp().Add(time.Duration(*vg.GetDeletionGracePeriodSeconds()) * time.Second)
		force := time.Now().After(forceAt)
		if force {
			logger.V(1).Info("force removal of volume group from host due to passed grace period")
		}
		if err := r.LVM.VGRemove(ctx, name, lvm2go.Force(force)); err != nil {
			if errors.Is(err, lvm2go.ErrVolumeGroupNotFound) {
				logger.V(1).Info("volume group not found on host, removing finalizer")
			} else {
				return ctrl.Result{}, fmt.Errorf("failed to remove volume group: %w", err)
			}
		}
		if updated := controllerutil.RemoveFinalizer(vg, VolumeGroupFinalizer); updated {
			return ctrl.Result{}, r.Update(ctx, vg)
		}
		logger.V(1).Info("volume group has been removed and is now waiting for other finalizers to be removed")
		return ctrl.Result{Requeue: true}, nil
	}

	logger.V(1).Info("syncing volume group with host, starting host discovery")
	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	lvm, err := r.LVM.VG(ctx, name, lvm2go.UnitBytes)

	logger.V(1).Info("host discovery completed", "duration", time.Since(start), "errorTypeFromLVMVG", fmt.Sprintf("%T", err), "errorValueFromLVMVG", err)

	if err != nil { // If there's any error from discovery
		logger.Info("Discovery returned an error", "errString", err.Error()) // Log the basic error string safely

		isNotFound := errors.Is(err, lvm2go.ErrVolumeGroupNotFound)
		logger.Info("Checked errors.Is(err, lvm2go.ErrVolumeGroupNotFound)", "isNotFound", isNotFound)

		var exitErr *exec.ExitError
		foundExecExitError := errors.As(err, &exitErr)
		logger.Info("Checked errors.As(err, &exitErr)", "foundExecExitError", foundExecExitError)

		isExitStatus5FromType := false
		if foundExecExitError && exitErr.ProcessState != nil {
			logger.Info("exec.ExitError found", "exitCode", exitErr.ProcessState.ExitCode())
			if exitErr.ProcessState.ExitCode() == 5 {
				isExitStatus5FromType = true
			}
		} else if foundExecExitError && exitErr.ProcessState == nil {
			logger.Info("exec.ExitError found, but ProcessState is nil")
		}

		// Fallback to string checking if typed checks fail
		errStr := err.Error() // Safe to call .Error() as err is not nil here
		containsExitStatus5InString := strings.Contains(errStr, "exit status 5")
		logger.Info("Checked strings.Contains(err.Error(), 'exit status 5')", "containsExitStatus5InString", containsExitStatus5InString, "errStr", errStr)

		if isNotFound || isExitStatus5FromType || (!isNotFound && !isExitStatus5FromType && containsExitStatus5InString) {
			logger.Info("Volume group considered not found or discovery failed with exit 5, attempting initialization.",
				"isNotFound", isNotFound,
				"isExitStatus5FromType", isExitStatus5FromType,
				"containsExitStatus5InString", containsExitStatus5InString,
				"originalError", err)
			err = r.initializeVG(ctx, vg) // Attempt to create it
		}
		// If err was not any of the above "not found" conditions, and it was not nil initially,
		// it will fall through to the next check. If initializeVG was called and failed, err will also be set here.
	}

	// After potential discovery error handling and/or initialization attempt:
	if err != nil {
		logger.Error(err, "Error after discovery and potential initialization phases")
		return ctrl.Result{}, errors.Join(err, r.Client.Status().Update(ctx, vg))
	}

	if updated := controllerutil.AddFinalizer(vg, VolumeGroupFinalizer); updated {
		return ctrl.Result{Requeue: true}, r.Update(ctx, vg)
	}

	if err = r.sync(ctx, vg, lvm); err != nil {
		err = fmt.Errorf("failed to sync volume group with lvm2: %w", err)
	}

	if statusErr := r.syncStatus(ctx, vg, lvm); statusErr != nil {
		err = errors.Join(err, fmt.Errorf("failed to sync status from lvm2 into volume group: %w", statusErr))
	} else {
		logger.V(1).Info("status refreshed successfully")
	}

	if err := errors.Join(err, r.Client.Status().Update(ctx, vg)); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: r.SyncInterval}, nil
}

func (r *VolumeGroupReconciler) initializeVG(ctx context.Context, vg *v1alpha1.VolumeGroup) error {
	start := time.Now()
	log.FromContext(ctx).Info("creating volume group on host")
	defer func() {
		log.FromContext(ctx).Info("finished creating volume group on host", "duration", time.Since(start))
	}()

	opts, err := convertToVGCreateOptions(ctx, vg)
	if err != nil {
		return fmt.Errorf("failed to convert VolumeGroup to VGCreateOptions: %w", err)
	}

	if err = r.LVM.VGCreate(ctx, opts); err != nil {
		SetSyncedOnHostCreationFailed(&vg.Status.Conditions, vg.GetGeneration(), err)
	}

	return err
}

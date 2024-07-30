package controller

import (
	"context"
	"fmt"

	"github.com/jakobmoellerdev/lvm2go"
	"github.com/topolvm/topovgm/api/v1alpha1"
	"github.com/topolvm/topovgm/internal/selector"
	"github.com/topolvm/topovgm/internal/utils"
	"k8s.io/apimachinery/pkg/api/resource"
)

func nameOnNode(vg *v1alpha1.VolumeGroup) lvm2go.VolumeGroupName {
	name := lvm2go.VolumeGroupName(vg.GetName())
	if vg.Spec.NameOnNode != nil {
		name = lvm2go.VolumeGroupName(*vg.Spec.NameOnNode)
	}
	return name
}

func pvsFromSpec(ctx context.Context, vg *v1alpha1.VolumeGroup) ([]lvm2go.PhysicalVolumeName, error) {
	desiredState := vg.Spec.PVs
	if len(desiredState) == 0 {
		if fromSelector, err := selector.DevicesMatchingSelector(ctx, vg.Spec.PVSelector); err != nil {
			return nil, fmt.Errorf("could not get devices matching selector: %w", err)
		} else {
			desiredState = fromSelector
		}
	}
	return utils.Map(desiredState, func(pv string) lvm2go.PhysicalVolumeName {
		return lvm2go.PhysicalVolumeName(pv)
	}), nil
}

func convertToVGCreateOptions(ctx context.Context, vg *v1alpha1.VolumeGroup) (*lvm2go.VGCreateOptions, error) {
	opts := &lvm2go.VGCreateOptions{
		VolumeGroupName: nameOnNode(vg),
	}

	if vg.Spec.Tags != nil {
		opts.Tags = vg.Spec.Tags
	}

	var err error
	opts.PhysicalVolumeNames, err = pvsFromSpec(ctx, vg)
	if err != nil {
		return nil, fmt.Errorf("could not get physical volume names from spec: %w", err)
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

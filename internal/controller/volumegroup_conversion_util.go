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

// getNameOnNode returns the VolumeGroupName based on the NameOnNode field in the VolumeGroup spec.
// If NameOnNode is not set, it falls back to using the UID of the VolumeGroup.
//
// Parameters:
// - vg: The VolumeGroup object containing the spec with the NameOnNode field.
//
// Returns:
// - The VolumeGroupName derived from either the NameOnNode field or the UID.
func getNameOnNode(vg *v1alpha1.VolumeGroup) lvm2go.VolumeGroupName {
	if vg.Spec.NameOnNode == nil {
		return lvm2go.VolumeGroupName(vg.GetUID())
	}
	return lvm2go.VolumeGroupName(*vg.Spec.NameOnNode)
}

// getPhysicalVolumeNames retrieves the physical volume names from the VolumeGroup spec based on the provided PhysicalVolumeSelector.
// It uses selector.DevicesMatchingSelector to get the devices matching the selector and maps them to PhysicalVolumeName.
//
// Parameters:
// - ctx: The context for the operation.
// - vg: The VolumeGroup object containing the spec with the PhysicalVolumeSelector.
//
// Returns:
// - A slice of PhysicalVolumeName containing the names of the physical volumes.
// - An error if there was an issue retrieving the devices matching the selector.
func getPhysicalVolumeNames(ctx context.Context, vg *v1alpha1.VolumeGroup) ([]lvm2go.PhysicalVolumeName, error) {
	fromSelector, err := selector.DevicesMatchingSelector(ctx, vg.Spec.PhysicalVolumeSelector)

	if err != nil {
		return nil, fmt.Errorf("could not get devices matching selector: %w", err)
	}

	return utils.Map(fromSelector, func(pv string) lvm2go.PhysicalVolumeName {
		return lvm2go.PhysicalVolumeName(pv)
	}), nil
}

func convertToVGCreateOptions(ctx context.Context, vg *v1alpha1.VolumeGroup) (*lvm2go.VGCreateOptions, error) {
	opts := &lvm2go.VGCreateOptions{
		VolumeGroupName: getNameOnNode(vg),
	}

	if vg.Spec.Tags != nil {
		opts.Tags = vg.Spec.Tags
	}

	var err error
	opts.PhysicalVolumeNames, err = getPhysicalVolumeNames(ctx, vg)
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

	if vols := vg.Spec.MaximumPhysicalVolumes; vols != nil {
		opts.MaximumPhysicalVolumes = lvm2go.MaximumPhysicalVolumes(*vols)
	}

	if vols := vg.Spec.MaximumLogicalVolumes; vols != nil {
		opts.MaximumLogicalVolumes = lvm2go.MaximumLogicalVolumes(*vols)
	}

	if vg.Spec.PhysicalExtentSize != nil {
		physicalExtentSize, err := convertQuantityToSize(vg.Spec.PhysicalExtentSize)
		if err != nil {
			return nil, err
		}
		opts.PhysicalExtentSize = lvm2go.PhysicalExtentSize(physicalExtentSize)
	}

	if vg.Spec.MetadataSize != nil {
		metadataSize, err := convertQuantityToSize(vg.Spec.MetadataSize)
		if err != nil {
			return nil, err
		}
		opts.MetadataSize = lvm2go.MetadataSize(metadataSize)
	}

	if vg.Spec.DataAlignment != nil {
		dataAlignment, err := convertQuantityToSize(vg.Spec.DataAlignment)
		if err != nil {
			return nil, err
		}
		opts.DataAlignment = lvm2go.DataAlignment(dataAlignment)
	}

	if vg.Spec.DataAlignmentOffset != nil {
		dataAlignment, err := convertQuantityToSize(vg.Spec.DataAlignmentOffset)
		if err != nil {
			return nil, err
		}
		opts.DataAlignmentOffset = lvm2go.DataAlignmentOffset(dataAlignment)
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

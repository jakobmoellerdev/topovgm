package selector

import (
	"context"
	"fmt"
	"slices"

	"github.com/topolvm/topovgm/api/v1alpha1"
	"github.com/topolvm/topovgm/internal/lsblk"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var runLSBLK = lsblk.LSBLK

func DevicesMatchingSelector(ctx context.Context, selector v1alpha1.PVSelector) ([]string, error) {
	if len(selector.PVSelectorTerms) == 0 {
		return nil, nil
	}

	columns := make([]lsblk.Column, 0, len(selector.PVSelectorTerms))
	columns = append(columns, lsblk.ColumnPath)
	for _, term := range selector.PVSelectorTerms {
		for _, requirement := range term.MatchLSBLK {
			columns = append(columns, lsblk.Column(requirement.Key))
		}
	}
	slices.Sort(columns)
	columns = slices.Compact(columns)

	devices, err := runLSBLK(ctx, columns...)
	if err != nil {
		return nil, fmt.Errorf("failed to list block devices for selector translation: %w", err)
	}
	devices = lsblk.RecursiveBlockDevices(devices)

	log.FromContext(ctx).V(1).Info("devices discovered from LSBLK", "count", len(devices))

	var selected []string

	for _, term := range selector.PVSelectorTerms {
		for _, dev := range devices {
			var matches int
			for _, requirement := range term.MatchLSBLK {
				if match, err := matchesLSBLKRequirement(dev, requirement); err != nil {
					return nil, fmt.Errorf("could not match requirement %v: %w", requirement, err)
				} else if match {
					matches++
				}
			}
			// If all requirements are met, add the device to the list of selected devices
			if matches == len(term.MatchLSBLK) {
				if kname, exists := dev.GetString(lsblk.ColumnPath); !exists {
					return nil, fmt.Errorf("block device %s is missing path", kname)
				} else {
					selected = append(selected, kname)
				}
			}
		}
	}

	slices.Sort(selected)
	// Remove duplicate matches
	selected = slices.Compact(selected)

	return selected, nil
}

func matchesLSBLKRequirement(dev lsblk.BlockDevice, requirement v1alpha1.LSBLKSelectorRequirement) (matches bool, err error) {
	switch requirement.Operator {
	case v1alpha1.PVSelectorOpExists:
		if _, ok := dev.Get(lsblk.Column(requirement.Key)); ok {
			matches = true
		}
	case v1alpha1.PVSelectorOpDoesNotExist:
		if _, ok := dev.Get(lsblk.Column(requirement.Key)); !ok {
			matches = true
		}
	case v1alpha1.PVSelectorOpIn:
		for _, v := range requirement.Values {
			if val, _ := dev.GetString(lsblk.Column(requirement.Key)); v == val {
				matches = true
				break
			}
		}
	case v1alpha1.PVSelectorGt:
		// The values array must have a single element, which will be interpreted as a resource.Quantity.
		// parse the value as a resource.Quantity
		var quantityRequirement resource.Quantity
		if quantityRequirement, err = resource.ParseQuantity(requirement.Values[0]); err != nil {
			return false, fmt.Errorf("value is not a valid quantity: %w", err)
		}

		// Get the value from the block device and parse it as an int
		var intFromLSBLK int
		fromLSBLK, ok := dev.Get(lsblk.Column(requirement.Key))
		if ok {
			intFromLSBLK, ok = fromLSBLK.(int)
		} else {
			return false, fmt.Errorf("column %s with value %v cannot be converted to int for comparison", requirement.Key, fromLSBLK)
		}
		// Then convert it to a resource.Quantity
		quantityFromLSBLK := resource.NewQuantity(int64(intFromLSBLK), resource.DecimalSI)

		// If the value is greater than the requirement, it matches.
		if quantityFromLSBLK.Cmp(quantityRequirement) == 1 {
			matches = true
		}
	}
	return
}

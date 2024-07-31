package selector

import (
	"context"
	"testing"

	"github.com/jakobmoellerdev/lvm2go"
	"github.com/topolvm/topovgm/api/v1alpha1"
	"github.com/topolvm/topovgm/internal/lsblk"
)

func TestDevicesMatchingSelector(t *testing.T) {
	device, err := lvm2go.NewLoopbackDevice(lvm2go.MustParseSize("10M"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := device.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	selector := v1alpha1.PhysicalVolumeSelector{{
		MatchLSBLK: []v1alpha1.LSBLKSelectorRequirement{
			{
				Key:      v1alpha1.LSBLKSelectorKey(lsblk.ColumnPath),
				Operator: v1alpha1.PVSelectorOpIn,
				Values:   []string{device.Device()},
			},
		},
	}}

	devices, err := DevicesMatchingSelector(context.Background(), selector)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("unexpected devices: %v", devices)
	}
}

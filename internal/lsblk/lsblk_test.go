package lsblk

import (
	"context"
	"testing"

	"github.com/jakobmoellerdev/lvm2go"
)

func TestLSBLK(t *testing.T) {
	device, err := lvm2go.NewLoopbackDevice(lvm2go.MustParseSize("10M"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := device.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	lsblk, err := LSBLK(context.Background(), ColumnName, ColumnKName)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for i := range RecursiveBlockDevices(lsblk) {
		if name, _ := lsblk[i].GetString(ColumnName); name == device.Device() {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("device %s not found in lsblk output", device.Device())
	}
}

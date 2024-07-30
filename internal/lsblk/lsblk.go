package lsblk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jakobmoellerdev/lvm2go"
	"github.com/topolvm/topovgm/internal/utils"
)

const lsblkCommand = "/usr/bin/lsblk"
const nsenterCommand = "/usr/bin/nsenter"

// Column is the type of key that can be used in a node selector requirement.
// +enum
type Column string

const (
	ColumnName         Column = "NAME"         // device name
	ColumnKName        Column = "KNAME"        // internal kernel device name
	ColumnPath         Column = "PATH"         // path to the device node
	ColumnMajMin       Column = "MAJ:MIN"      // major:minor device number
	ColumnFSAvail      Column = "FSAVAIL"      // filesystem size available
	ColumnFSSize       Column = "FSSIZE"       // filesystem size
	ColumnFSType       Column = "FSTYPE"       // filesystem type
	ColumnFSUsed       Column = "FSUSED"       // filesystem size used
	ColumnFSUsePerc    Column = "FSUSE%"       // filesystem use percentage
	ColumnFSRoots      Column = "FSROOTS"      // mounted filesystem roots
	ColumnFSVer        Column = "FSVER"        // filesystem version
	ColumnMountPoint   Column = "MOUNTPOINT"   // where the device is mounted
	ColumnMountPoints  Column = "MOUNTPOINTS"  // all locations where device is mounted
	ColumnLabel        Column = "LABEL"        // filesystem LABEL
	ColumnUUID         Column = "UUID"         // filesystem UUID
	ColumnPTUUID       Column = "PTUUID"       // partition table identifier (usually UUID)
	ColumnPTType       Column = "PTTYPE"       // partition table type
	ColumnPartType     Column = "PARTTYPE"     // partition type code or UUID
	ColumnPartTypeName Column = "PARTTYPENAME" // partition type name
	ColumnPartLabel    Column = "PARTLABEL"    // partition LABEL
	ColumnPartUUID     Column = "PARTUUID"     // partition UUID
	ColumnPartFlags    Column = "PARTFLAGS"    // partition flags
	ColumnRA           Column = "RA"           // read-ahead of the device
	ColumnRO           Column = "RO"           // read-only device
	ColumnRM           Column = "RM"           // removable device
	ColumnHotplug      Column = "HOTPLUG"      // removable or hotplug device (usb, pcmcia, ...)
	ColumnModel        Column = "MODEL"        // device identifier
	ColumnSerial       Column = "SERIAL"       // disk serial number
	ColumnSize         Column = "SIZE"         // size of the device
	ColumnState        Column = "STATE"        // state of the device
	ColumnOwner        Column = "OWNER"        // user name
	ColumnGroup        Column = "GROUP"        // group name
	ColumnMode         Column = "MODE"         // device node permissions
	ColumnAlignment    Column = "ALIGNMENT"    // alignment offset
	ColumnMinIO        Column = "MIN-IO"       // minimum I/O size
	ColumnOptIO        Column = "OPT-IO"       // optimal I/O size
	ColumnPhySec       Column = "PHY-SEC"      // physical sector size
	ColumnLogSec       Column = "LOG-SEC"      // logical sector size
	ColumnRota         Column = "ROTA"         // rotational device
	ColumnSched        Column = "SCHED"        // I/O scheduler name
	ColumnRQSize       Column = "RQ-SIZE"      // request queue size
	ColumnType         Column = "TYPE"         // device type
	ColumnDiscAln      Column = "DISC-ALN"     // discard alignment offset
	ColumnDiscGran     Column = "DISC-GRAN"    // discard granularity
	ColumnDiscMax      Column = "DISC-MAX"     // discard max bytes
	ColumnDiscZero     Column = "DISC-ZERO"    // discard zeroes data
	ColumnWSame        Column = "WSAME"        // write same max bytes
	ColumnWWN          Column = "WWN"          // unique storage identifier
	ColumnRand         Column = "RAND"         // adds randomness
	ColumnPKName       Column = "PKNAME"       // internal parent kernel device name
	ColumnHCTL         Column = "HCTL"         // Host:Channel:Target:Lun for SCSI
	ColumnTran         Column = "TRAN"         // device transport type
	ColumnSubsystems   Column = "SUBSYSTEMS"   // de-duplicated chain of subsystems
	ColumnRev          Column = "REV"          // device revision
	ColumnVendor       Column = "VENDOR"       // device vendor
	ColumnZoned        Column = "ZONED"        // zone model
	ColumnDAX          Column = "DAX"          // dax-capable device
)

type BlockDevice map[string]any

func (dev BlockDevice) GetString(col Column) (string, bool) {
	val, ok := dev.Get(col)
	if !ok {
		return "", false
	}
	return val.(string), true
}

func (dev BlockDevice) Get(col Column) (any, bool) {
	val, ok := dev[strings.ToLower(string(col))]
	return val, ok
}

func (dev BlockDevice) Children() []BlockDevice {
	children, ok := dev["children"]
	if !ok {
		return []BlockDevice{}
	}
	return utils.Map(children.([]any), func(t any) BlockDevice {
		return t.(map[string]any)
	})
}

// LSBLK lists the block devices using the lsblk command with the provided columns
func LSBLK(ctx context.Context, columns ...Column) ([]BlockDevice, error) {
	columnsOption := strings.Join(utils.Map(columns, func(t Column) string {
		return string(t)
	}), ",")
	// var output bytes.Buffer
	var blockDeviceMap map[string][]BlockDevice
	args := []string{"--json", "--bytes", "-o", columnsOption}

	if err := runInto(ctx, &blockDeviceMap, args...); err != nil {
		return []BlockDevice{}, err
	}

	return blockDeviceMap["blockdevices"], nil
}

// runInto calls sub-commands and decodes the output via JSON into the provided struct pointer.
// if the struct pointer is nil, the output will be printed to the log instead.
func runInto(ctx context.Context, into *map[string][]BlockDevice, args ...string) error {
	var cmd *exec.Cmd

	if lvm2go.IsContainerized(ctx) {
		args = append([]string{"-m", "-u", "-i", "-n", "-p", "-t", "1", lsblkCommand}, args...)
		cmd = exec.CommandContext(ctx, nsenterCommand, args...)
	} else {
		cmd = exec.CommandContext(ctx, lsblkCommand, args...)
	}
	cmd.Env = append(cmd.Env, "LC_ALL=C")

	output, err := lvm2go.StreamedCommand(ctx, cmd)
	if err != nil {
		return fmt.Errorf("failed to execute command: %v", err)
	}

	return errors.Join(json.NewDecoder(output).Decode(&into), output.Close())
}

func RecursiveBlockDevices(devices []BlockDevice) (devicesWithChildren []BlockDevice) {
	for _, dev := range devices {
		devicesWithChildren = append(devicesWithChildren, dev)
		devicesWithChildren = append(devicesWithChildren, RecursiveBlockDevices(dev.Children())...)
	}
	return
}

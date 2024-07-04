package methods

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

// JAILER CONFIGURATION
func ExampleJailerConfig_enablingJailer() {
	const socketPath = "api.socket"
	ctx := context.Background()
	vmmCtx, vmmCancel := context.WithCancel(ctx)
	defer vmmCancel()

	const id = "4569"
	//
	const kernelImagePath = "../vmlinux-5.10.210"

	uid := 123
	gid := 100

	fcCfg := firecracker.Config{
		SocketPath:      socketPath,
		KernelImagePath: kernelImagePath,
		KernelArgs:      "console=ttyS0 reboot=k panic=1 pci=off",
		Drives:          firecracker.NewDrivesBuilder("../ubuntu-22.04.ext4.3").Build(),
		LogLevel:        "Debug",
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  firecracker.Int64(1),
			Smt:        firecracker.Bool(false),
			MemSizeMib: firecracker.Int64(256),
		},
		JailerCfg: &firecracker.JailerConfig{
			UID:            &uid,
			GID:            &gid,
			ID:             id,
			NumaNode:       firecracker.Int(0),
			JailerBinary:   "../jailer",
			ChrootBaseDir:  "/srv/jailer",
			CgroupVersion:  "2",
			ChrootStrategy: firecracker.NewNaiveChrootStrategy(kernelImagePath),
			ExecFile:       "../firecracker",
		},
	}

	// Check if kernel image is readable
	f, err := os.Open(fcCfg.KernelImagePath)
	if err != nil {
		panic(fmt.Errorf("failed to open kernel image: %v", err))
	}
	f.Close()

	// Check each drive is readable and writable
	for _, drive := range fcCfg.Drives {
		drivePath := firecracker.StringValue(drive.PathOnHost)
		f, err := os.OpenFile(drivePath, os.O_RDWR, 0666)
		if err != nil {
			panic(fmt.Errorf("failed to open drive with read/write permissions: %v", err))
		}
		f.Close()
	}

	m, err := firecracker.NewMachine(vmmCtx, fcCfg)
	if err != nil {
		log.Println(err)
		panic(err)
	}

	if err := m.Start(vmmCtx); err != nil {
		log.Println(err)
		panic(err)
	}
	defer m.StopVMM()

	// wait for the VMM to exit
	if err := m.Wait(vmmCtx); err != nil {
		log.Println(err)
		panic(err)
	}
}

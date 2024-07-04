package methods

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

func ExampleNetworkInterface_rateLimiting() {
	const socketPath = "/tmp/firecracker.sock"
	// construct the limitations of the bandwidth for firecracker
	bandwidthBuilder := firecracker.TokenBucketBuilder{}.
		WithInitialSize(1024 * 1024).        // Initial token amount
		WithBucketSize(1024 * 1024).         // Max number of tokens
		WithRefillDuration(30 * time.Second) // Refill rate

	// construct the limitations of the number of operations per duration for firecracker
	opsBuilder := firecracker.TokenBucketBuilder{}.
		WithInitialSize(5).
		WithBucketSize(5).
		WithRefillDuration(5 * time.Second)

	// create the inbound rate limiter.
	inbound := firecracker.NewRateLimiter(bandwidthBuilder.Build(), opsBuilder.Build())

	bandwidthBuilder = bandwidthBuilder.WithBucketSize(1024 * 1024 * 10)
	opsBuilder = opsBuilder.
		WithBucketSize(100).
		WithInitialSize(100)
	// create the outbound rate limiter
	outbound := firecracker.NewRateLimiter(bandwidthBuilder.Build(), opsBuilder.Build())

	// network interface with static configuration
	// Hardcoded the data.
	networkIfaces := []firecracker.NetworkInterface{{
		StaticConfiguration: &firecracker.StaticNetworkConfiguration{
			MacAddress:  "AA:FC:00:00:00:01",
			HostDevName: "tap0",
			IPConfiguration: &firecracker.IPConfiguration{
				IPAddr: net.IPNet{
					IP:   net.IPv4(172, 16, 0, 2),
					Mask: net.IPMask{255, 255, 255, 0},
				},
				Gateway:     net.IPv4(172, 16, 0, 1),
				Nameservers: []string{"8.8.8.8"},
				IfName:      "eth0",
			},
		},
		InRateLimiter:  inbound,
		OutRateLimiter: outbound,
	}}

	// config file for the firecracker process.
	cfg := firecracker.Config{
		SocketPath:      "api.socket",
		KernelImagePath: "../vmlinux-5.10.210",
		Drives:          firecracker.NewDrivesBuilder("../ubuntu-22.04.ext4.3").Build(),
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  firecracker.Int64(1),
			MemSizeMib: firecracker.Int64(1024),
		},
		NetworkInterfaces: networkIfaces,
	}

	// creating paths for the standard error and standard outpout path.
	stdOutPath := "/tmp/stdout.log"
	stdout, err := os.OpenFile(stdOutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Errorf("failed to create stdout file: %v", err))
	}
	stdErrPath := "/tmp/stderr.log"
	stderr, err := os.OpenFile(stdErrPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Errorf("failed to create stderr file: %v", err))
	}

	ctx := context.Background()

	cmd := firecracker.VMCommandBuilder{}.
		WithBin("../firecracker").
		WithSocketPath(socketPath).
		WithStdout(stdout).
		WithStderr(stderr).
		Build(ctx)

	m, err := firecracker.NewMachine(ctx, cfg, firecracker.WithProcessRunner(cmd))
	if err != nil {
		panic(fmt.Errorf("failed to create new machine: %v", err))
	}

	defer os.Remove(cfg.SocketPath)

	if err := m.Start(ctx); err != nil {
		panic(fmt.Errorf("failed to initialize machine: %v", err))
	}
	info, err := m.DescribeInstanceInfo(ctx)
	if err != nil {
		fmt.Println("Unable to fetch the VM Info: ", err)
	}
	fmt.Println(*info.State, *info.ID, *info.AppName)
	// wait for VMM to execute
	if err := m.Wait(ctx); err != nil {
		panic(err)
	}
}

package methods

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/weaveworks/ignite/pkg/logs"
)

func CreateContainer(name string) (string, error) {
	options := network.CreateOptions{
		Driver: "bridge",
		Labels: map[string]string{
			"mylabel": "value",
		},
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet:  "172.16.0.0/24",
					Gateway: "172.16.0.1",
				},
			},
		},
		Options: map[string]string{},
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logs.Logger.Errorf("Failed to create Docker client: %v", err)
		return "", err
	}
	networkName := "testJailer"
	networkResp, err := cli.NetworkCreate(context.Background(), networkName, options)
	if err != nil {
		log.Fatalf("Failed to create network: %v", err)
		return "", err
	}
	logs.Logger.Infof("Created network: %s\n", networkName)

	containerName := name
	containerConfig := &container.Config{
		Image: "alpine",
		Cmd:   []string{"sh", "-c", "sleep infinity"},
	}
	hostConfig := &container.HostConfig{
		AutoRemove:  true,
		NetworkMode: container.NetworkMode(networkResp.ID),
	} 

	// Create the container
	containerResp, err := cli.ContainerCreate(
		context.Background(),
		containerConfig,
		hostConfig,
		nil,
		nil,
		containerName, // Name for the container
	)
	if err != nil {
		logs.Logger.Errorf("Failed to create a container: %v", err)
		return "", err
	}
	logs.Logger.Infof("Created container: %s\n", containerResp.ID)

	if err := cli.ContainerStart(context.Background(), containerResp.ID, container.StartOptions{}); err != nil {
		logs.Logger.Errorf("Failed to start container: %v", err)
		return "", err
	}
	fmt.Printf("Started container: %s\n", containerResp.ID)

	pid, err := getContainerPID(containerResp.ID)
	if err != nil {
		logs.Logger.Errorf("Failed to get container PID: %v", err)
		return "", err
	}
	netnsPath := fmt.Sprintf("/proc/%d/ns/net", pid)
	return netnsPath, nil
}

func getContainerPID(containerID string) (int, error) {
	cmd := exec.Command("docker", "inspect", "--format", "{{.State.Pid}}", containerID)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	outputStr := strings.TrimSpace(string(output))
	pid := 0
	fmt.Sscanf(outputStr, "%d", &pid)
	return pid, nil
}

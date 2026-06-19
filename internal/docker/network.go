package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/network"
)

// NetworkName is the name of the isolated Docker bridge network used by Magicbox.
const NetworkName = "magicbox_net"

// EnsureNetwork checks if the magicbox_net network exists and creates it if not.
// The network is created with inter-container communication disabled to enforce
// isolation between app containers.
func (c *Client) EnsureNetwork(ctx context.Context) error {
	networks, err := c.cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return fmt.Errorf("docker: failed to list networks: %w", err)
	}

	for _, n := range networks {
		if n.Name == NetworkName {
			return nil
		}
	}

	_, err = c.cli.NetworkCreate(ctx, NetworkName, network.CreateOptions{
		Driver: "bridge",
		Options: map[string]string{
			"com.docker.network.bridge.enable_icc": "true",
		},
	})
	if err != nil {
		return fmt.Errorf("docker: failed to create network %s: %w", NetworkName, err)
	}

	return nil
}

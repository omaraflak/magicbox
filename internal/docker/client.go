// Package docker provides a Docker client wrapper for Magicbox container management.
package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/client"
)

// Client wraps the Docker SDK client.
type Client struct {
	cli *client.Client
}

// New creates a new Docker client connected to the local Docker daemon
// via unix:///var/run/docker.sock with automatic API version negotiation.
// It verifies connectivity with a Ping (10s timeout).
func New() (*Client, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("docker: failed to create client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := cli.Ping(ctx); err != nil {
		cli.Close()
		return nil, fmt.Errorf("docker: daemon not reachable: %w", err)
	}

	return &Client{cli: cli}, nil
}

// Close releases the Docker client resources.
func (c *Client) Close() error {
	return c.cli.Close()
}

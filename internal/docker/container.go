package docker

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/go-connections/nat"
)

// ContainerPrefix is the naming prefix for all Magicbox app containers.
const ContainerPrefix = "magicbox_app_"

// ContainerName returns the canonical container name for a user's app instance.
func ContainerName(username, appID string) string {
	return ContainerPrefix + username + "_" + appID
}

// AppVolumeMount describes a shared volume to mount into a container.
type AppVolumeMount struct {
	Name     string
	ReadOnly bool
}

// AppContainerConfig holds all parameters needed to create an app container.
// This is a local type to avoid importing the core package.
type AppContainerConfig struct {
	AppID        string
	AppName      string
	Image        string
	EntryPort    int
	RouteSlug    string
	Username     string
	UserID       string
	AppToken     string
	CoreURL      string // e.g., "magicbox_core:50051"
	MagicboxRoot string // e.g., "/opt/magicbox"
	VolumeMounts []AppVolumeMount
	MemoryMB     int
	CPUCores     float64
	Host         string
}

// CreateAndStartContainer creates and starts a new app container with the given configuration.
// It sets up environment variables, volume binds, Traefik labels, resource limits,
// connects the container to the magicbox_net network, and starts it.
func (c *Client) CreateAndStartContainer(ctx context.Context, cfg *AppContainerConfig) (string, error) {
	// 1. Build environment variables.
	env := []string{
		"MAGICBOX_API_TOKEN=" + cfg.AppToken,
		"MAGICBOX_CORE_URL=" + cfg.CoreURL,
		"MAGICBOX_USER_ID=" + cfg.UserID,
		"MAGICBOX_APP_ID=" + cfg.AppID,
	}

	// 2. Build volume binds.
	binds := []string{
		// Private app state directory.
		filepath.Join(cfg.MagicboxRoot, "users", cfg.Username, "apps", cfg.AppID) + ":/data/app_state:rw",
		// Transit directory (shared scratch space).
		filepath.Join(cfg.MagicboxRoot, "transit") + ":/data/transit:rw",
	}
	for _, vm := range cfg.VolumeMounts {
		access := "rw"
		if vm.ReadOnly {
			access = "ro"
		}
		hostPath := filepath.Join(cfg.MagicboxRoot, "users", cfg.Username, "shared", vm.Name)
		binds = append(binds, hostPath+":/data/shared/"+vm.Name+":"+access)
	}

	// 3. Disable direct Traefik routing to enforce Core gateway auth
	labels := map[string]string{}

	// 4. Exposed port.
	portStr := fmt.Sprintf("%d/tcp", cfg.EntryPort)
	exposedPorts := nat.PortSet{
		nat.Port(portStr): struct{}{},
	}

	// 5. Resource limits.
	var resources container.Resources
	if cfg.MemoryMB > 0 {
		resources.Memory = int64(cfg.MemoryMB) * 1024 * 1024
	}
	if cfg.CPUCores > 0 {
		resources.NanoCPUs = int64(cfg.CPUCores * 1e9)
	}

	// 6. Create the container.
	name := ContainerName(cfg.Username, cfg.AppID)

	// Clean up any container that already exists with the same name to prevent naming conflicts.
	if exists, existingID, _, err := c.ContainerExistsByName(ctx, name); err == nil && exists {
		_ = c.RemoveContainer(ctx, existingID)
	}

	resp, err := c.cli.ContainerCreate(ctx,
		&container.Config{
			Image:        cfg.Image,
			Env:          env,
			ExposedPorts: exposedPorts,
			Labels:       labels,
		},
		&container.HostConfig{
			Binds:     binds,
			Resources: resources,
		},
		nil,
		nil,
		name,
	)
	if err != nil {
		return "", fmt.Errorf("docker: failed to create container %s: %w", name, err)
	}

	// 7. Connect to magicbox_net.
	if err := c.cli.NetworkConnect(ctx, NetworkName, resp.ID, nil); err != nil {
		return "", fmt.Errorf("docker: failed to connect container %s to network: %w", name, err)
	}

	// 8. Start the container.
	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("docker: failed to start container %s: %w", name, err)
	}

	return resp.ID, nil
}

// StopContainer stops a running container with the given timeout in seconds.
func (c *Client) StopContainer(ctx context.Context, containerID string, timeoutSeconds int) error {
	timeout := timeoutSeconds
	return c.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

// RemoveContainer removes a container by ID, forcing removal if necessary.
func (c *Client) RemoveContainer(ctx context.Context, containerID string) error {
	return c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
}

// ContainerStatus holds the runtime state of a container.
type ContainerStatus struct {
	Running   bool
	IPAddress string
	ExitCode  int
}

// InspectContainer returns the status of a container by ID.
func (c *Client) InspectContainer(ctx context.Context, containerID string) (*ContainerStatus, error) {
	info, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("docker: failed to inspect container %s: %w", containerID, err)
	}

	status := &ContainerStatus{
		Running:  info.State.Running,
		ExitCode: info.State.ExitCode,
	}

	if netSettings, ok := info.NetworkSettings.Networks[NetworkName]; ok {
		status.IPAddress = netSettings.IPAddress
	}

	return status, nil
}

// PullImage pulls a Docker image and returns its digest.
// If force is false, it implements an "IfNotPresent" strategy: if the image exists locally,
// it skips pulling and returns the local image ID.
func (c *Client) PullImage(ctx context.Context, img string, force bool) (string, error) {
	if !force {
		if id, err := c.getLocalImageID(ctx, img); err == nil {
			return id, nil
		}
	}

	reader, err := c.cli.ImagePull(ctx, img, image.PullOptions{})
	if err != nil {
		// Fallback to local image if remote pull fails (e.g. for local-only developer images)
		if id, localErr := c.getLocalImageID(ctx, img); localErr == nil {
			return id, nil
		}
		return "", fmt.Errorf("docker: failed to pull image %s: %w", img, err)
	}
	defer reader.Close()

	// Read the entire pull output to ensure it completes.
	output, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("docker: failed to read pull response for %s: %w", img, err)
	}

	// Extract digest from the pull output.
	// The Docker daemon includes "Digest: sha256:..." in the output.
	digest := extractDigest(string(output))
	return digest, nil
}

func (c *Client) getLocalImageID(ctx context.Context, img string) (string, error) {
	inspect, _, err := c.cli.ImageInspectWithRaw(ctx, img)
	if err != nil {
		return "", err
	}
	return inspect.ID, nil
}

// extractDigest parses a "Digest: sha256:..." string from Docker pull output.
func extractDigest(output string) string {
	const prefix = "Digest: "
	for _, line := range strings.Split(output, "\n") {
		if idx := strings.Index(line, prefix); idx >= 0 {
			digest := strings.TrimSpace(line[idx+len(prefix):])
			// Clean up any trailing JSON artifacts.
			if end := strings.IndexAny(digest, "\"} \t"); end > 0 {
				digest = digest[:end]
			}
			return digest
		}
	}
	return ""
}

// ContainerExistsByName checks if a container with the given name exists.
// It returns whether the container exists, its ID, and whether it is currently running.
func (c *Client) ContainerExistsByName(ctx context.Context, name string) (bool, string, bool, error) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("name", "^/"+name+"$")),
	})
	if err != nil {
		return false, "", false, fmt.Errorf("docker: failed to list containers: %w", err)
	}

	for _, ctr := range containers {
		for _, n := range ctr.Names {
			// Docker prepends "/" to container names.
			if n == "/"+name {
				running := ctr.State == "running"
				return true, ctr.ID, running, nil
			}
		}
	}

	return false, "", false, nil
}

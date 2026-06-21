package docker

import (
	"context"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

// TraefikContainerName is the name of the Traefik reverse proxy container.
const TraefikContainerName = "magicbox_traefik"

// TraefikImage is the Docker image used for the Traefik reverse proxy.
const TraefikImage = "traefik:v3.6.1"

// EnsureTraefik makes sure the Traefik reverse proxy container is running.
// If the container exists but is stopped, it starts it.
// If the container does not exist, it creates and starts it.
func (c *Client) EnsureTraefik(ctx context.Context) error {
	exists, containerID, running, err := c.ContainerExistsByName(ctx, TraefikContainerName)
	if err != nil {
		return fmt.Errorf("docker: failed to check traefik container: %w", err)
	}

	if exists && running {
		return nil
	}

	if exists && !running {
		if err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
			return fmt.Errorf("docker: failed to start traefik container: %w", err)
		}
		return nil
	}

	// Pull image if not present locally.
	_, _, err = c.cli.ImageInspectWithRaw(ctx, TraefikImage)
	if err != nil {
		if _, err := c.PullImage(ctx, TraefikImage, false); err != nil {
			return fmt.Errorf("docker: failed to pull traefik image: %w", err)
		}
	}

	exposedPorts, portBindings, err := nat.ParsePortSpecs([]string{"80:80/tcp"})
	if err != nil {
		return fmt.Errorf("docker: failed to parse traefik port spec: %w", err)
	}

	resp, err := c.cli.ContainerCreate(ctx,
		&container.Config{
			Image: TraefikImage,
			Env:   []string{"DOCKER_API_VERSION=1.40"},
			Cmd: []string{
				"--providers.docker=true",
				"--providers.docker.exposedbydefault=false",
				"--providers.docker.network=" + NetworkName,
				"--entrypoints.web.address=:80",
			},
			ExposedPorts: exposedPorts,
		},
		&container.HostConfig{
			Binds:        []string{"/var/run/docker.sock:/var/run/docker.sock:ro"},
			PortBindings: portBindings,
			RestartPolicy: container.RestartPolicy{
				Name: container.RestartPolicyUnlessStopped,
			},
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				NetworkName: {},
			},
		},
		nil,
		TraefikContainerName,
	)
	if err != nil {
		return fmt.Errorf("docker: failed to create traefik container: %w", err)
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("docker: failed to start traefik container: %w", err)
	}

	return nil
}

// GenerateTraefikLabels returns the Docker labels needed for Traefik to route
// traffic to an app container based on the user's username and the app's route slug or custom domain.
func GenerateTraefikLabels(username, routeSlug string, entryPort int, customHost string) map[string]string {
	routerName := username + "-" + routeSlug

	labels := map[string]string{
		"traefik.enable": "true",
		"traefik.http.routers." + routerName + ".entrypoints": "web",
		"traefik.http.services." + routerName + ".loadbalancer.server.port": strconv.Itoa(entryPort),
	}

	if customHost != "" {
		labels["traefik.http.routers."+routerName+".rule"] = "Host(`" + customHost + "`)"
	} else {
		prefix := "/u/" + username + "/" + routeSlug
		labels["traefik.http.routers."+routerName+".rule"] = "PathPrefix(`" + prefix + "`)"
		labels["traefik.http.routers."+routerName+".middlewares"] = routerName + "-strip"
		labels["traefik.http.middlewares."+routerName+"-strip.stripprefix.prefixes"] = prefix
	}

	return labels
}

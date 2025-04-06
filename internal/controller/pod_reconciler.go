package controller

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/Kraftnetes/k8s-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// reconcilePod creates a Pod for the GameServer resource by splitting the work into neat helper functions.
// It checks for an existing Pod, builds the container specs, applies configuration overrides,
// and finally creates the Pod while setting proper owner references.
func (r *GameServerReconciler) reconcilePod(ctx context.Context, gs *v1alpha1.GameServer, gameDef *v1alpha1.GameDefinition) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	id := ResolveGameServerId(gs)
	podName := fmt.Sprintf("gs-%s-pod", id)
	pvcName := fmt.Sprintf("gs-%s-pvc", id)

	// Check if the Pod already exists.
	pod, err := r.getExistingPod(ctx, podName, gs.Namespace)
	if err != nil {
		// Something unexpected happened while trying to get the Pod.
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to get Pod")
			r.Recorder.Event(gs, corev1.EventTypeWarning, "PodLookupFailed", err.Error())
			return ctrl.Result{}, err
		}
	}
	if pod != nil {
		// Pod already exists; nothing more to do.
		return ctrl.Result{}, nil
	}

	// Resolve configuration and environment details.
	mergedConfig, finalEnv, finalResources := resolveConfigEnvResources(gs, gameDef)

	// Process container ports; note that logger is passed by value (not as a pointer).
	containerPorts := resolveContainerPorts(&mergedConfig, logger)

	// Build the primary game container.
	gameContainer := buildGameContainer(mergedConfig, finalEnv, finalResources, containerPorts, gameDef.Spec.Storage.Enabled.BoolVal)

	// Start assembling the containers list.
	containers := []corev1.Container{gameContainer}

	// Determine if the file browser should be enabled.
	fileBrowserEnabled := gameDef.Spec.FileBrowser.BoolVal
	if gs.Spec.Filebrowser != nil {
		fileBrowserEnabled = *gs.Spec.Filebrowser
	}
	if fileBrowserEnabled {
		containers = append(containers, buildFileBrowserContainer())
	}

	// Build volumes if storage is enabled.
	volumes := buildPodVolumes(pvcName, gameDef.Spec.Storage.Enabled.BoolVal)

	// Build the Pod specification.
	podSpec := buildPodSpec(containers, volumes)

	// Create the Pod object.
	pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: gs.Namespace,
			Labels: map[string]string{
				"app":        "gameserver",
				"gameserver": gs.Name,
				"kraftnetes-id": gs.Labels["kraftnetes-id"],
			},
		},
		Spec: podSpec,
	}

	// Set GameServer as the owner of the Pod.
	if err := controllerutil.SetControllerReference(gs, pod, r.Scheme); err != nil {
		r.Recorder.Event(gs, corev1.EventTypeWarning, "OwnerRefError", err.Error())
		return ctrl.Result{}, err
	}

	// Create the Pod.
	if err := r.Create(ctx, pod); err != nil {
		logger.Error(err, "Failed to create Pod")
		r.Recorder.Event(gs, corev1.EventTypeWarning, "PodCreateFailed", err.Error())
		return ctrl.Result{}, err
	}

	r.Recorder.Eventf(gs, corev1.EventTypeNormal, "PodCreated", "Created Pod %s", pod.Name)
	logger.Info("Created Pod", "name", pod.Name)
	return ctrl.Result{}, nil
}

// getExistingPod checks if a Pod with the given name exists.
func (r *GameServerReconciler) getExistingPod(ctx context.Context, podName, namespace string) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: namespace}, pod)
	if err != nil {
		return nil, err
	}
	return pod, nil
}

// buildGameContainer returns the container spec for the game server.
func buildGameContainer(mergedConfig v1alpha1.GameDefinitionSpec, finalEnv []corev1.EnvVar, finalResources corev1.ResourceRequirements, containerPorts []corev1.ContainerPort, storageEnabled bool) corev1.Container {
	container := corev1.Container{
		Name:      "game-server",
		Image:     mergedConfig.Image,
		TTY:       true,
		Stdin:     true,
		Ports:     containerPorts,
		Env:       finalEnv,
		Resources: finalResources,
	}
	if storageEnabled {
		container.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "game-data",
				MountPath: "/data",
			},
		}
	}
	return container
}

// buildFileBrowserContainer returns the container spec for the file browser.
func buildFileBrowserContainer() corev1.Container {
	return corev1.Container{
		Name:  "filebrowser",
		Image: "filebrowser/filebrowser",
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 8077,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Args: []string{"--port", "8077"},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "game-data",
				MountPath: "/srv",
			},
		},
	}
}

// buildPodVolumes returns the volumes spec for the Pod if storage is enabled.
func buildPodVolumes(pvcName string, storageEnabled bool) []corev1.Volume {
	if storageEnabled {
		return []corev1.Volume{
			{
				Name: "game-data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
				},
			},
		}
	}
	return nil
}

// buildPodSpec returns the PodSpec from given containers and volumes.
func buildPodSpec(containers []corev1.Container, volumes []corev1.Volume) corev1.PodSpec {
	return corev1.PodSpec{
		Containers: containers,
		Volumes:    volumes,
	}
}

// resolveContainerPorts processes container ports: if a port's type is "HostPort", assign a random host port.
func resolveContainerPorts(mergedConfig *v1alpha1.GameDefinitionSpec, logger logr.Logger) []corev1.ContainerPort {
	var containerPorts []corev1.ContainerPort
	for _, gp := range mergedConfig.Ports {
		cp := corev1.ContainerPort{
			ContainerPort: gp.ContainerPort.IntVal,
			Name:          gp.Name,
			Protocol:      corev1.Protocol(gp.Protocol),
		}
		if gp.Type == "HostPort" {
			hostPort := resolveHostPort()
			logger.Info("Attaching host port", "hostPort", hostPort)
			cp.HostPort = hostPort
		}
		containerPorts = append(containerPorts, cp)
	}
	return containerPorts
}

// resolveConfigEnvResources merges the GameDefinition and GameServer configurations,
// applying profile overrides and merging environment variables.
func resolveConfigEnvResources(gs *v1alpha1.GameServer, gameDef *v1alpha1.GameDefinition) (v1alpha1.GameDefinitionSpec, []corev1.EnvVar, corev1.ResourceRequirements) {
	mergedConfig := gameDef.Spec

	// Determine which profile to apply.
	profileName := gs.Spec.Profile
	if profileName == "" && gameDef.Spec.Profiles != nil && gameDef.Spec.Profiles.Default != "" {
		profileName = gameDef.Spec.Profiles.Default
	}

	var chosenProfile *v1alpha1.GameProfile
	if profileName != "" && gameDef.Spec.Profiles != nil {
		// Avoid the common pitfall of taking the address of the loop variable.
		for i := range gameDef.Spec.Profiles.Values {
			if gameDef.Spec.Profiles.Values[i].Name == profileName {
				chosenProfile = &gameDef.Spec.Profiles.Values[i]
				break
			}
		}
	}

	// If a profile was found, patch the base configuration with its settings.
	if chosenProfile != nil {
		if chosenProfile.Image != "" {
			mergedConfig.Image = chosenProfile.Image
		}
		mergedConfig.FileBrowser = chosenProfile.FileBrowser
		if chosenProfile.StopStrategy != nil {
			mergedConfig.StopStrategy = chosenProfile.StopStrategy
		}
		if chosenProfile.RestartStrategy != nil {
			mergedConfig.RestartStrategy = chosenProfile.RestartStrategy
		}
		if chosenProfile.Storage != nil {
			mergedConfig.Storage = chosenProfile.Storage
		}
		if len(chosenProfile.Ports) > 0 {
			mergedConfig.Ports = chosenProfile.Ports
		}
		mergedConfig.Env = mergeEnvVars(mergedConfig.Env, chosenProfile.Env)
	}

	// Finally, override with GameServer-specific settings.
	finalEnv := mergeEnvVars(mergedConfig.Env, gs.Spec.Env)
	finalResources := gs.Spec.Resources
	return mergedConfig, finalEnv, finalResources
}

// resolveHostPort returns a random host port in the range [30000, 33332].
func resolveHostPort() int32 {
	return rand.Int31n(3333) + 30000
}

// mergeEnvVars merges two slices of corev1.EnvVar.
// Variables in the override slice take precedence over those in the base slice.
func mergeEnvVars(base, override []corev1.EnvVar) []corev1.EnvVar {
	envMap := make(map[string]corev1.EnvVar)
	for _, env := range base {
		envMap[env.Name] = env
	}
	for _, env := range override {
		envMap[env.Name] = env
	}
	merged := make([]corev1.EnvVar, 0, len(envMap))
	for _, env := range envMap {
		merged = append(merged, env)
	}
	return merged
}

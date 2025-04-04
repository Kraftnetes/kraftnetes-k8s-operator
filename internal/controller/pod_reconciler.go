package controller

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/Kraftnetes/k8s-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// reconcilePod creates a Pod for the GameServer resource.
// It fetches the base configuration from the GameDefinition named in gs.Spec.Game,
// optionally applies profile overrides (merging environment variables),
// attaches a volume mount, and finally patches in values from the GameServer.
func (r *GameServerReconciler) reconcilePod(ctx context.Context, gs *v1alpha1.GameServer, gameDef *v1alpha1.GameDefinition) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	podName := fmt.Sprintf("gs-%s-pod", gs.Name)
	pvcName := fmt.Sprintf("gs-%s-pvc", gs.Name)

	// Check if the Pod already exists.
	pod := &corev1.Pod{}
	if err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: gs.Namespace}, pod); err == nil {
		// Pod already exists; nothing more to do.
		return ctrl.Result{}, nil
	} else if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "Failed to get Pod")
		r.Recorder.Event(gs, corev1.EventTypeWarning, "PodLookupFailed", err.Error())
		return ctrl.Result{}, err
	}

	// Start with the base configuration from GameDefinition.
	mergedConfig := gameDef.Spec

	// Determine which profile to apply.
	// Profile is optional in GameServer. If provided use it, otherwise, use the default profile from GameDefinition if defined.
	profileName := gs.Spec.Profile
	if profileName == "" && gameDef.Spec.Profiles != nil && gameDef.Spec.Profiles.Default != "" {
		profileName = gameDef.Spec.Profiles.Default
	}

	var chosenProfile *v1alpha1.GameProfile
	if profileName != "" && gameDef.Spec.Profiles != nil {
		for _, p := range gameDef.Spec.Profiles.Values {
			if p.Name == profileName {
				chosenProfile = &p
				break
			}
		}
	}

	// If a profile was found, patch the base configuration with its settings.
	// For most fields we override, but for Env we merge.
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
		// Merge profile Env with the base Env.
		mergedConfig.Env = mergeEnvVars(mergedConfig.Env, chosenProfile.Env)
	}

	// Finally, override with GameServer-specific settings.
	// Merge environment variables from GameServer.
	finalEnv := mergeEnvVars(mergedConfig.Env, gs.Spec.Env)
	finalResources := gs.Spec.Resources

	// Process container ports: if a port's type is HostPort, assign a random host port.
	var containerPorts []corev1.ContainerPort
	for _, gp := range mergedConfig.Ports {
		cp := corev1.ContainerPort{
			ContainerPort: gp.ContainerPort.IntVal,
			Name:          gp.Name,
			Protocol:      corev1.Protocol(gp.Protocol),
		}
		if gp.Type == "HostPort" {
			cp.HostPort = resolveHostPort()
		}
		containerPorts = append(containerPorts, cp)
	}

	// ---start pod def
	gameContainer := corev1.Container{
		Name:      "game-server",
		Image:     mergedConfig.Image,
		TTY:       true,
		Stdin:     true,
		Ports:     containerPorts,
		Env:       finalEnv,
		Resources: finalResources,
	}

	if gameDef.Spec.Storage.Enabled.BoolVal {
		gameContainer.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "game-data",
				MountPath: "/data",
			},
		}
	}

	podSpec := corev1.PodSpec{
		Containers: []corev1.Container{gameContainer},
	}

	fileBrowserEnabled := gameDef.Spec.FileBrowser.BoolVal
	if gs.Spec.Filebrowser != nil {
		fileBrowserEnabled = *gs.Spec.Filebrowser
	}

	if fileBrowserEnabled {

		fileBrowserContainer := corev1.Container{
			Name:  "filebrowser",
			Image: "filebrowser/filebrowser",
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: 8077,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			Args: []string{
				"--port", "8077",
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "game-data",
					MountPath: "/srv",
				},
			},
		}

		podSpec.Containers = append(podSpec.Containers, fileBrowserContainer)
	}

	if gameDef.Spec.Storage.Enabled.BoolVal {
		podSpec.Volumes = []corev1.Volume{
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

	pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: gs.Namespace,
			Labels: map[string]string{
				"app":        "gameserver",
				"gameserver": gs.Name,
			},
		},
		Spec: podSpec,
	}
	// ---end pod def

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

/*


// Create the Pod definition.
	pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: gs.Namespace,
			Labels: map[string]string{
				"app":        "gameserver",
				"gameserver": gs.Name,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:      "game-server",
					Image:     mergedConfig.Image,
					TTY:       true,
					Stdin:     true,
					Ports:     containerPorts,
					Env:       finalEnv,
					Resources: finalResources,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "game-data",
							MountPath: "/data",
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "game-data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
		},
	}


*/

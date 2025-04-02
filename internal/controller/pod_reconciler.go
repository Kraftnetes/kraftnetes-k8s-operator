package controller

import (
	"context"
	"fmt"

	"github.com/Kraftnetes/k8s-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *GameServerReconciler) reconcilePod(ctx context.Context, gs *v1alpha1.GameServer) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	podName := fmt.Sprintf("gs-%s-pod", gs.Name)
	pod := &corev1.Pod{}
	if err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: gs.Namespace}, pod); err == nil {
		return ctrl.Result{}, nil
	} else if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "Failed to get Pod")
		r.Recorder.Event(gs, corev1.EventTypeWarning, "PodLookupFailed", err.Error())
		return ctrl.Result{}, err
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
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "game-server",
					Image: "itzg/minecraft-server:latest",
					TTY:   true,
					Stdin: true,
					Ports: []corev1.ContainerPort{{
						ContainerPort: 25565,
						Name:          "minecraft",
						Protocol:      corev1.ProtocolTCP,
					}},
					Env:       gs.Spec.Env,
					Resources: gs.Spec.Resources,
					VolumeMounts: []corev1.VolumeMount{{
						Name:      "game-data",
						MountPath: "/data",
					}},
				},
			},
			Volumes: []corev1.Volume{{
				Name: "game-data",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			}},
		},
	}

	if err := controllerutil.SetControllerReference(gs, pod, r.Scheme); err != nil {
		r.Recorder.Event(gs, corev1.EventTypeWarning, "OwnerRefError", err.Error())
		return ctrl.Result{}, err
	}

	if err := r.Create(ctx, pod); err != nil {
		logger.Error(err, "Failed to create Pod")
		r.Recorder.Event(gs, corev1.EventTypeWarning, "PodCreateFailed", err.Error())
		return ctrl.Result{}, err
	}

	r.Recorder.Eventf(gs, corev1.EventTypeNormal, "PodCreated", "Created Pod %s", pod.Name)
	logger.Info("Created Pod", "name", pod.Name)
	return ctrl.Result{}, nil
}

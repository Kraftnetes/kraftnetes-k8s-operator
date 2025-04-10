package controller

import (
	"context"
	"fmt"

	"github.com/Kraftnetes/k8s-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *GameServerReconciler) reconcilePvc(ctx context.Context, gs *v1alpha1.GameServer, gameDef *v1alpha1.GameDefinition) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !gameDef.Spec.Storage.Enabled.BoolVal {
		logger.Info("Storage disabled for %s. Skipping...", gameDef.Name)
		r.Recorder.Eventf(gs, corev1.EventTypeNormal, "Skipped creating pvc for %s", gs.Name)
		return ctrl.Result{}, nil
	}

	id := ResolveGameServerId(gs)
	pvcName := fmt.Sprintf("gs-%s-pvc", id)

	pvc := &corev1.PersistentVolumeClaim{}
	if err := r.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: gs.Namespace}, pvc); err == nil {
		return ctrl.Result{}, nil
	} else if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "Failed to get pvc")
		r.Recorder.Event(gs, corev1.EventTypeWarning, "PvcLookupFailed", err.Error())
		return ctrl.Result{}, err
	}

	defaultStorage := "10Gi" //hard coded default storage. can be overriden by game definition or game server

	if gameDef.Spec.Storage.DefaultSize != "" {
		defaultStorage = gameDef.Spec.Storage.DefaultSize
	}

	if gs.Spec.VolumeSize != "" {
		defaultStorage = gs.Spec.VolumeSize
	}

	pvc = &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: gs.Namespace,
			Labels: map[string]string{
				"app":        "gameserver",
				"gameserver": gs.Name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(defaultStorage),
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(gs, pvc, r.Scheme); err != nil {
		r.Recorder.Event(gs, corev1.EventTypeWarning, "OwnerRefError", err.Error())
		return ctrl.Result{}, err
	}

	if err := r.Create(ctx, pvc); err != nil {
		logger.Error(err, "Failed to create Pvc")
		r.Recorder.Event(gs, corev1.EventTypeWarning, "PvcCreateFailed", err.Error())
		return ctrl.Result{}, err
	}

	r.Recorder.Eventf(gs, corev1.EventTypeNormal, "PvcCreated", "Created Pvc %s", pvc.Name)
	logger.Info("Created Pvc", "name", pvc.Name)
	return ctrl.Result{}, nil
}

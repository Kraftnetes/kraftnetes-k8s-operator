package controller

import (
	"context"
	"fmt"

	"github.com/Kraftnetes/k8s-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *GameServerReconciler) reconcileService(ctx context.Context, gs *v1alpha1.GameServer, gameDef *v1alpha1.GameDefinition) (ctrl.Result, error) {

	logger := log.FromContext(ctx)

	fileBrowserEnabled := gameDef.Spec.FileBrowser.BoolVal
	if gs.Spec.Filebrowser != nil {
		fileBrowserEnabled = *gs.Spec.Filebrowser
	}
	if !fileBrowserEnabled {
		logger.Info("File browser disabled. Skipping creation of filebrowser service.", "gameName", gameDef.Name)
		r.Recorder.Eventf(gs, corev1.EventTypeNormal, "SkippedFileBrowserService", "Filebrowser service is disabled for %s", gs.Name)
		return ctrl.Result{}, nil
	}

	id := ResolveGameServerId(gs)
	serviceName := fmt.Sprintf("gs-%s-filebrowser-service", id)

	service := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: gs.Namespace}, service); err == nil {
		return ctrl.Result{}, nil
	} else if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "Failed to get Service")
		r.Recorder.Event(gs, corev1.EventTypeWarning, "ServiceLookupFailed", err.Error())
		return ctrl.Result{}, err
	}

	service = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: gs.Namespace,
			Labels: map[string]string{
				"app":        "filebrowser",
				"gameserver": gs.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":        "gameserver",
				"gameserver": gs.Name,
			},
			Ports: []corev1.ServicePort{{
				Name:       "filebrowser",
				Port:       8080,
				TargetPort: intstr.FromInt(8077),
				Protocol:   corev1.ProtocolTCP,
			}},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	if err := controllerutil.SetControllerReference(gs, service, r.Scheme); err != nil {
		r.Recorder.Event(gs, corev1.EventTypeWarning, "OwnerRefError", err.Error())
		return ctrl.Result{}, err
	}

	if err := r.Create(ctx, service); err != nil {
		logger.Error(err, "Failed to create Service")
		r.Recorder.Event(gs, corev1.EventTypeWarning, "ServiceCreateFailed", err.Error())
		return ctrl.Result{}, err
	}

	r.Recorder.Eventf(gs, corev1.EventTypeNormal, "ServiceCreated", "Created Service %s", service.Name)
	logger.Info("Created Service", "name", service.Name)
	return ctrl.Result{}, nil
}

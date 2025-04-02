package controller

import (
	"context"
	"reflect"

	"github.com/Kraftnetes/k8s-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *GameServerReconciler) updateStatus(ctx context.Context, gs *v1alpha1.GameServer) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	desired := gs.DeepCopy()
	desired.Status.State = "Running"
	desired.Status.Message = "Pod is active"

	if reflect.DeepEqual(gs.Status, desired.Status) {
		return ctrl.Result{}, nil // No update needed
	}

	if err := r.Status().Update(ctx, desired); err != nil {
		logger.Error(err, "Failed to update GameServer status")
		return ctrl.Result{}, err
	}

	r.Recorder.Event(gs, "Normal", "StatusUpdated", "Updated status to Running")
	return ctrl.Result{}, nil
}

func (r *GameServerReconciler) reconcileInitialStatus(ctx context.Context, gs *v1alpha1.GameServer) (ctrl.Result, error) {
	if gs.Status.State != "" {
		return ctrl.Result{}, nil
	}

	gs.Status.State = "Pending"
	gs.Status.Message = "GameServer is pending initialization"
	if err := r.Status().Update(ctx, gs); err != nil {
		return ctrl.Result{}, err
	}
	r.Recorder.Event(gs, "Normal", "Initializing", "GameServer is pending initialization")
	return ctrl.Result{}, nil
}

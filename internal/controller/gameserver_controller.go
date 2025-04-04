package controller

import (
	"context"
	"fmt"

	"github.com/Kraftnetes/k8s-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type GameServerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=kraftnetes.com,resources=gameservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kraftnetes.com,resources=gameservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kraftnetes.com,resources=gameservers/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods;services,verbs=get;list;watch;create;update;patch;delete

func (r *GameServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	gameServer := &v1alpha1.GameServer{}
	if err := r.Get(ctx, req.NamespacedName, gameServer); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	gameDef := &v1alpha1.GameDefinition{}
	if err := r.Get(ctx, types.NamespacedName{Name: gameServer.Spec.Game}, gameDef); err != nil {
		logger.Error(err, "GameDefinition not found", "game", gameServer.Spec.Game)
		r.Recorder.Event(gameServer, corev1.EventTypeWarning, "GameDefinitionNotFound", fmt.Sprintf("GameDefinition %s not found", gameServer.Spec.Game))
		// Requeue until the GameDefinition is available.
		return ctrl.Result{Requeue: true}, nil
	}

	for _, sub := range r.subReconcilers() {
		if res, err := sub(ctx, gameServer, gameDef); err != nil || !res.IsZero() {
			if err != nil {
				logger.Error(err, "Subreconciler failed")
				r.Recorder.Event(gameServer, corev1.EventTypeWarning, "SubreconcileError", err.Error())
			}
			return res, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *GameServerReconciler) subReconcilers() []func(context.Context, *v1alpha1.GameServer, *v1alpha1.GameDefinition) (ctrl.Result, error) {
	return []func(context.Context, *v1alpha1.GameServer, *v1alpha1.GameDefinition) (ctrl.Result, error){
		r.reconcileInitialStatus,
		r.reconcileService,
		r.reconcilePvc,
		r.reconcilePod,
		r.updateStatus,
	}
}

func (r *GameServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("gameserver-controller")

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.GameServer{}).
		Owns(&corev1.Pod{}).
		//Owns(&corev1.Service{}).
		Complete(r)
}

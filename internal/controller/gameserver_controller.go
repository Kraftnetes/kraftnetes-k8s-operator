package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kraftnetescomv1alpha1 "github.com/Kraftnetes/k8s-operator/api/v1alpha1"
)

type GameServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kraftnetes.com,resources=gameservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kraftnetes.com,resources=gameservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kraftnetes.com,resources=gameservers/finalizers,verbs=update

func (r *GameServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the GameServer instance
	gameServer := &kraftnetescomv1alpha1.GameServer{}
	err := r.Get(ctx, req.NamespacedName, gameServer)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			logger.Info("GameServer not found, ignoring")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to fetch GameServer")
		return ctrl.Result{}, err
	}

	podName := "gs-" + gameServer.Name + "-pod"
	serviceName := "gs-" + gameServer.Name + "-service"

	// Check if the Pod exists
	pod := &corev1.Pod{}
	err = r.Get(ctx, types.NamespacedName{Name: podName, Namespace: gameServer.Namespace}, pod)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "Failed to check for existing Pod")
		return ctrl.Result{}, err
	}

	if err != nil {
		// Pod doesn't exist — create it
		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: gameServer.Namespace,
				Labels: map[string]string{
					"app":        "gameserver",
					"gameserver": gameServer.Name,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "game-server",
						Image: "itzg/minecraft-server:latest",
						TTY:   true,
						Stdin: true,
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 25565,
								Name:          "minecraft",
								Protocol:      corev1.ProtocolTCP,
							},
						},
						Env:       gameServer.Spec.Env,
						Resources: gameServer.Spec.Resources,
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
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
		}

		// ✅ Set controller reference
		if err := controllerutil.SetControllerReference(gameServer, pod, r.Scheme); err != nil {
			logger.Error(err, "Failed to set controller reference on Pod")
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, pod); err != nil {
			logger.Error(err, "Failed to create Pod")
			return ctrl.Result{}, err
		}
		logger.Info("Created Pod for GameServer", "Pod", podName)
	}

	// Check if the Service exists
	service := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: gameServer.Namespace}, service)
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "Failed to check for existing Service")
		return ctrl.Result{}, err
	}

	if err != nil {
		// Service doesn't exist — create it
		service = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: gameServer.Namespace,
				Labels: map[string]string{
					"app":        "gameserver",
					"gameserver": gameServer.Name,
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app":        "gameserver",
					"gameserver": gameServer.Name,
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "minecraft",
						Port:       25565,
						TargetPort: intstr.FromInt(25565),
						Protocol:   corev1.ProtocolTCP,
						NodePort:   0, // Let K8s assign
					},
				},
				Type: corev1.ServiceTypeNodePort,
			},
		}

		// ✅ Set controller reference
		if err := controllerutil.SetControllerReference(gameServer, service, r.Scheme); err != nil {
			logger.Error(err, "Failed to set controller reference on Service")
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, service); err != nil {
			logger.Error(err, "Failed to create Service")
			return ctrl.Result{}, err
		}
		logger.Info("Created NodePort Service for GameServer", "Service", serviceName)
	}

	// Update GameServer status
	gameServer.Status.State = "Running"
	gameServer.Status.Message = "Pod is active"
	if err := r.Status().Update(ctx, gameServer); err != nil {
		logger.Error(err, "Failed to update GameServer status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *GameServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kraftnetescomv1alpha1.GameServer{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

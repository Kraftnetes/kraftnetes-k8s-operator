package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Kraftnetes/k8s-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
		r.Recorder.Event(gameServer, corev1.EventTypeWarning, "GameDefinitionNotFound",
			fmt.Sprintf("GameDefinition %s not found", gameServer.Spec.Game))
		// Requeue until the GameDefinition is available.
		return ctrl.Result{Requeue: true}, nil
	}

	// --- VARIABLE SUBSTITUTION SECTION ---
	// Walk through gameDef.Spec and replace every occurrence of &{variable}
	// with its corresponding value provided in gameServer.Spec.Inputs or,
	// if absent, the default in gameDef.Inputs. If any required variable is missing,
	// or if any placeholder remains unresolved, return an error.
	resolvedSpec, err := resolveGameDefinitionSpec(gameDef.Spec, gameServer.Spec.Inputs, gameDef.Inputs)
	if err != nil {
		logger.Error(err, "Failed to resolve GameDefinition spec variables")
		r.Recorder.Event(gameServer, corev1.EventTypeWarning, "UnresolvedVariables", err.Error())
		return ctrl.Result{}, err
	}
	// Replace the GameDefinition spec with the fully resolved version.
	gameDef.Spec = resolvedSpec
	// --- END VARIABLE SUBSTITUTION SECTION ---

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

// resolveGameDefinitionSpec performs a generic substitution on the entire GameDefinitionSpec.
// It replaces every occurrence of &{variable} with the corresponding value from gsInputs or, if absent,
// from defInputs (which contains default values). If a variable is required but neither provided nor defaulted,
// or if any placeholder remains, an error is returned.
func resolveGameDefinitionSpec(spec v1alpha1.GameDefinitionSpec, gsInputs map[string]apiextensionsv1.JSON, defInputs map[string]v1alpha1.GameDefinitionInput) (v1alpha1.GameDefinitionSpec, error) {
	// Marshal the spec to a JSON string.
	rawBytes, err := json.Marshal(spec)
	if err != nil {
		return spec, fmt.Errorf("failed to marshal GameDefinition spec: %w", err)
	}
	specStr := string(rawBytes)

	// Build substitution map.
	subs := make(map[string]string)
	// Loop over each input declared in the GameDefinition.
	for key, input := range defInputs {
		var value string
		if gsInputs != nil {
			if j, ok := gsInputs[key]; ok {
				value = jsonValueToString(j)
			}
		}
		// --- If no value was provided by the GameServer, use the default if available.
		if value == "" {
			var defaultStr string
			switch input.Type {
			case "string":
					// Expect a default string.
					if input.Default.Type == v1alpha1.AnyValString {
							defaultStr = input.Default.StrVal
					} else {
							// If the default is not a string, fall back to its generic conversion.
							defaultStr = input.Default.String()
					}
			case "number":
					if input.Default.Type == v1alpha1.AnyValNumber {
							defaultStr = strconv.FormatFloat(input.Default.NumVal, 'f', -1, 64)
					} else if input.Default.Type == v1alpha1.AnyValString {
							// Try to interpret the string as a number.
							if num, err := strconv.ParseFloat(input.Default.StrVal, 64); err == nil {
									defaultStr = strconv.FormatFloat(num, 'f', -1, 64)
							} else {
									return spec, fmt.Errorf("default value for variable %q is not a valid number", key)
							}
					} else {
							return spec, fmt.Errorf("default value for variable %q is not a valid number", key)
					}
			case "boolean":
					if input.Default.Type == v1alpha1.AnyValBool {
							defaultStr = strconv.FormatBool(input.Default.BoolVal)
					} else if input.Default.Type == v1alpha1.AnyValString {
							// Try to interpret the string as a boolean.
							if b, err := strconv.ParseBool(input.Default.StrVal); err == nil {
									defaultStr = strconv.FormatBool(b)
							} else {
									return spec, fmt.Errorf("default value for variable %q is not a valid boolean", key)
							}
					} else {
							return spec, fmt.Errorf("default value for variable %q is not a valid boolean", key)
					}
			default:
					// Fallback: use AnyVal’s generic String() method.
					defaultStr = input.Default.String()
			}
			if defaultStr == "" {
					return spec, fmt.Errorf("variable %q is required but not provided and no default is available", key)
			}
			value = defaultStr
	}
	
		// ---
		subs[key] = value
	}

	// Perform substitution for all placeholders of the form &{variable}.
	for key, value := range subs {
		placeholder := "&{" + key + "}"
		specStr = strings.ReplaceAll(specStr, placeholder, value)
	}

	// Check if any unresolved placeholder remains.
	if strings.Contains(specStr, "&{") {
		return spec, fmt.Errorf("unresolved variables remain in GameDefinition spec: %s", specStr)
	}

	// Unmarshal the resolved JSON back into a GameDefinitionSpec.
	var resolvedSpec v1alpha1.GameDefinitionSpec
	if err := json.Unmarshal([]byte(specStr), &resolvedSpec); err != nil {
		return spec, fmt.Errorf("failed to unmarshal resolved GameDefinition spec: %w", err)
	}
	return resolvedSpec, nil
}

// jsonValueToString converts an apiextensionsv1.JSON value to its string representation.
func jsonValueToString(j apiextensionsv1.JSON) string {
	var v interface{}
	if err := json.Unmarshal(j.Raw, &v); err != nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
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
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}

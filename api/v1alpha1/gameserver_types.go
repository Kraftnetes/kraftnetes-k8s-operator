package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GameServerSpec defines the desired state of GameServer
type GameServerSpec struct {
	Game        string                          `json:"game"`
	Profile     string                          `json:"profile,omitempty"`
	Inputs      map[string]apiextensionsv1.JSON `json:"inputs,omitempty"`
	VolumeSize  string                          `json:"volumeSize,omitempty"`
	Filebrowser *bool                           `json:"filebrowser,omitempty"`
	Console     *bool                           `json:"console,omitempty"`
	Env         []corev1.EnvVar                 `json:"env,omitempty"`
	Resources   corev1.ResourceRequirements     `json:"resources,omitempty"`
}

// GameServerStatus defines the observed state of GameServer
type GameServerStatus struct {
	State   string `json:"state,omitempty"`
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// GameServer is the Schema for the gameservers API
type GameServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GameServerSpec   `json:"spec,omitempty"`
	Status GameServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GameServerList contains a list of GameServer
type GameServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GameServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GameServer{}, &GameServerList{})
}

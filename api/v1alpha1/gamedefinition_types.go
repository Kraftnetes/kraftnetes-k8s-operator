/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GameDefinitionSpec defines the desired state of GameDefinition
type GameDefinitionSpec struct {
	Game            string           `json:"game"`
	Image           string           `json:"image"`
	FileBrowser     bool             `json:"filebrowser,omitempty"`
	StopStrategy    *StopStrategy    `json:"stopStrategy,omitempty"`
	RestartStrategy *RestartStrategy `json:"restartStrategy,omitempty"`
	Storage         *StorageConfig   `json:"storage,omitempty"`
	Ports           []GamePort       `json:"ports,omitempty"`
	Env             []corev1.EnvVar  `json:"env,omitempty"`
	Profiles        *GameProfiles    `json:"profiles,omitempty"`
}

// StopStrategy controls how the game server is shut down
type StopStrategy struct {
	Stdin               string   `json:"stdin,omitempty"`
	Cmd                 []string `json:"cmd,omitempty"`
	ShutdownGracePeriod string   `json:"shutdownGracePeriod,omitempty"`
}

// RestartStrategy controls how to restart the server (e.g., via in-game command)
type RestartStrategy struct {
	Cmd []string `json:"cmd,omitempty"`
}

// StorageConfig describes persistent storage options
type StorageConfig struct {
	Enabled     bool   `json:"enabled"`
	DefaultSize string `json:"defaultSize"`
}

// GamePort defines networking ports
type GamePort struct {
	Name          string `json:"name"`
	ContainerPort int32  `json:"containerPort"`
	Protocol      string `json:"protocol"`
	Type          string `json:"type"` // HostPort | NodePort | ClusterIP
}

// GameProfiles allows optional predefined profiles

type GameProfiles struct {
	Default string        `json:"default"`
	Values  []GameProfile `json:"values,omitempty"`
}

type GameProfile struct {
	Name            string           `json:"name"`
	Image           string           `json:"image,omitempty"`
	FileBrowser     bool             `json:"filebrowser,omitempty" default:"false"`
	StopStrategy    *StopStrategy    `json:"stopStrategy,omitempty"`
	RestartStrategy *RestartStrategy `json:"restartStrategy,omitempty"`
	Storage         *StorageConfig   `json:"storage,omitempty"`
	Ports           []GamePort       `json:"ports,omitempty"`
	Env             []corev1.EnvVar  `json:"env,omitempty"`
}

// GameDefinitionStatus defines the observed state of GameDefinition
type GameDefinitionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status

// GameDefinition is the Schema for the gamedefinitions API
type GameDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GameDefinitionSpec   `json:"spec,omitempty"`
	Status GameDefinitionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GameDefinitionList contains a list of GameDefinition
type GameDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GameDefinition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GameDefinition{}, &GameDefinitionList{})
}

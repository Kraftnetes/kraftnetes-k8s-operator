package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GameDefinitionInput defines an input variable for a GameDefinition.
// The Default field can be a string, number, or bool.
type GameDefinitionInput struct {
	Required    bool                 `json:"required,omitempty" default:"false"`
	Default     apiextensionsv1.JSON `json:"default,omitempty"`
	Description string               `json:"description,omitempty"`
	Type        string               `json:"type,omitempty" default:"string"`
}

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
	DefaultSize string `json:"defaultSize" default:"10Gi"`
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

// GameProfile defines an override profile for a GameDefinition.
type GameProfile struct {
	Name            string           `json:"name"`
	Image           string           `json:"image,omitempty"`
	FileBrowser     bool             `json:"filebrowser,omitempty" default:"false"` //this is probably a bug which would disable file browser
	StopStrategy    *StopStrategy    `json:"stopStrategy,omitempty"`
	RestartStrategy *RestartStrategy `json:"restartStrategy,omitempty"`
	Storage         *StorageConfig   `json:"storage,omitempty"`
	Ports           []GamePort       `json:"ports,omitempty"`
	Env             []corev1.EnvVar  `json:"env,omitempty"`
}

// GameDefinitionStatus defines the observed state of GameDefinition
type GameDefinitionStatus struct {
	// Additional status fields can be defined here.
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status

// GameDefinition is the Schema for the gamedefinitions API.
type GameDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Inputs defines the input variables for the GameDefinition.
	Inputs map[string]GameDefinitionInput `json:"inputs,omitempty"`
	Spec   GameDefinitionSpec             `json:"spec,omitempty"`
	Status GameDefinitionStatus           `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GameDefinitionList contains a list of GameDefinition.
type GameDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GameDefinition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GameDefinition{}, &GameDefinitionList{})
}

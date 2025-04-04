package v1alpha1

import (
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GameDefinitionInput defines an input variable for a GameDefinition.
type GameDefinitionInput struct {
    Required    bool   `json:"required,omitempty" default:"false"`
    Default     AnyVal `json:"default,omitempty"` // see note below
    Description string `json:"description,omitempty"`
    Type        string `json:"type,omitempty" default:"string"`
}

// AnyVal is just a placeholder for your defaults. 
// If you need to store typed default values (bool/string/int), you can do 
// something like an interface or your own custom union type. For the sake 
// of example, we keep it simple as a raw string. If you want to allow number, 
// bool, etc., you can define a more complex type that uses json.Unmarshaler 
// or apiextensionsv1.JSON. (It depends on how you handle defaults.)
type AnyVal struct {
    Value string `json:"value,omitempty"`
}

// GamePort defines networking ports. 
// We use IntOrString for containerPort so that user can specify "25565" or 25565
type GamePort struct {
    Name          string       `json:"name"`
    ContainerPort IntOrString  `json:"containerPort"` // can be int or string
    Protocol      string       `json:"protocol,omitempty"`
    Type          string       `json:"type,omitempty"` // HostPort | NodePort | etc.
}

// StopStrategy controls how the game server is shut down
type StopStrategy struct {
    Stdin               string   `json:"stdin,omitempty"`
    Cmd                 []string `json:"cmd,omitempty"`
    ShutdownGracePeriod string   `json:"shutdownGracePeriod,omitempty"`
}

// RestartStrategy controls how to restart the server
type RestartStrategy struct {
    Cmd []string `json:"cmd,omitempty"`
}

// StorageConfig describes persistent storage options
type StorageConfig struct {
    Enabled     BoolOrString   `json:"enabled"`
    DefaultSize string `json:"defaultSize" default:"10Gi"`
}

// GameProfile defines an override profile for a GameDefinition.
// Notice we use BoolOrString for filebrowser, so it can be `true` or `"somePlaceholder"`
type GameProfile struct {
    Name            string           `json:"name"`
    Image           string           `json:"image,omitempty"`
    FileBrowser     BoolOrString     `json:"filebrowser,omitempty"`
    StopStrategy    *StopStrategy    `json:"stopStrategy,omitempty"`
    RestartStrategy *RestartStrategy `json:"restartStrategy,omitempty"`
    Storage         *StorageConfig   `json:"storage,omitempty"`
    Ports           []GamePort       `json:"ports,omitempty"`
    Env             []corev1.EnvVar  `json:"env,omitempty"`
}

// GameProfiles allows optional predefined profiles
type GameProfiles struct {
    Default string         `json:"default,omitempty"`
    Values  []GameProfile  `json:"values,omitempty"`
}

// GameDefinitionSpec defines the desired state of GameDefinition
type GameDefinitionSpec struct {
    Game            string           `json:"game"`
    Image           string           `json:"image"`
    FileBrowser     BoolOrString     `json:"filebrowser,omitempty"` // can be bool or string
    StopStrategy    *StopStrategy    `json:"stopStrategy,omitempty"`
    RestartStrategy *RestartStrategy `json:"restartStrategy,omitempty"`
    Storage         *StorageConfig   `json:"storage,omitempty"`
    Ports           []GamePort       `json:"ports,omitempty"`
    Env             []corev1.EnvVar  `json:"env,omitempty"`
    Profiles        *GameProfiles    `json:"profiles,omitempty"`
}

// GameDefinitionStatus defines the observed state of GameDefinition
type GameDefinitionStatus struct {
    // ...
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status

// GameDefinition is the Schema for the gamedefinitions API.
type GameDefinition struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

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

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SlackConfigSpec defines the desired state of SlackConfig
type SlackConfigSpec struct {
	// AuthType specifies the authentication type: "Webhook" or "Token" (Slack App).
	// +kubebuilder:validation:Enum=Webhook;Token
	AuthType string `json:"authType"`

	// WebhookUrlSecretRef references a Secret containing the Webhook URL. Required if AuthType is Webhook.
	// +optional
	WebhookURLSecretRef *corev1.SecretKeySelector `json:"webhookUrlSecretRef,omitempty"`

	// TokenSecretRef references a Secret containing the Slack OAuth Token. Required if AuthType is Token.
	// +optional
	TokenSecretRef *corev1.SecretKeySelector `json:"tokenSecretRef,omitempty"`

	// Channel is the default channel to send notifications to.
	// +optional
	Channel string `json:"channel,omitempty"`
}

// SlackConfigStatus defines the observed state of SlackConfig.
type SlackConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the SlackConfig resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SlackConfig is the Schema for the slackconfigs API
type SlackConfig struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of SlackConfig
	// +required
	Spec SlackConfigSpec `json:"spec"`

	// status defines the observed state of SlackConfig
	// +optional
	Status SlackConfigStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// SlackConfigList contains a list of SlackConfig
type SlackConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []SlackConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SlackConfig{}, &SlackConfigList{})
}

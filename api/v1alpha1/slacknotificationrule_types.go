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

// SlackNotificationRuleSpec defines the desired state of SlackNotificationRule
type SlackNotificationRuleSpec struct {
	// TargetResource specifies the resource kind to watch.
	// +kubebuilder:validation:Enum=CronJob;CronWorkflow
	TargetResource string `json:"targetResource"`

	// LabelSelector selects the resources to be monitored.
	LabelSelector metav1.LabelSelector `json:"labelSelector"`

	// SlackConfigRef references the SlackConfig to use.
	SlackConfigRef corev1.LocalObjectReference `json:"slackConfigRef"`

	// Notifications defines the rules for sending notifications.
	Notifications []NotificationRule `json:"notifications"`
}

type NotificationRule struct {
	// Status is the resource status that triggers the notification (e.g., Running, Succeeded, Failed).
	Status string `json:"status"`

	// Title is the title template to send.
	// +optional
	Title string `json:"title,omitempty"`

	// Message is the message template to send.
	Message string `json:"message"`

	// Color specifies the attachment color (e.g., "good", "warning", "danger", "#ff0000").
	// +optional
	Color string `json:"color,omitempty"`

	// Channel overrides the default channel in SlackConfig.
	// +optional
	Channel string `json:"channel,omitempty"`
}

// SlackNotificationRuleStatus defines the observed state of SlackNotificationRule.
type SlackNotificationRuleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the SlackNotificationRule resource.
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

// SlackNotificationRule is the Schema for the slacknotificationrules API
type SlackNotificationRule struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of SlackNotificationRule
	// +required
	Spec SlackNotificationRuleSpec `json:"spec"`

	// status defines the observed state of SlackNotificationRule
	// +optional
	Status SlackNotificationRuleStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// SlackNotificationRuleList contains a list of SlackNotificationRule
type SlackNotificationRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []SlackNotificationRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SlackNotificationRule{}, &SlackNotificationRuleList{})
}

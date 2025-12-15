package controller

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	notificationv1alpha1 "github.com/murasame29/slack-notifier-controller/api/v1alpha1"
	"github.com/murasame29/slack-notifier-controller/internal/slack"
)

const (
	AnnotationSentStatuses = "notification.murasame29.com/sent-statuses"
)

type Notifier struct {
	Client      client.Client
	SlackClient slack.Client
}

// Notify checks rules and sends notifications.
// triggerObj: The object that triggered the event (e.g., Job, Workflow)
// targetObj: The object that rules target (e.g., CronJob, CronWorkflow)
func (n *Notifier) Notify(ctx context.Context, triggerObj client.Object, targetObj client.Object, status string) error {
	logger := log.FromContext(ctx)

	// List all Rules
	var ruleList notificationv1alpha1.SlackNotificationRuleList
	if err := n.Client.List(ctx, &ruleList); err != nil {
		return fmt.Errorf("failed to list notification rules: %w", err)
	}

	// Filter rules that match the target object
	var matchedRules []notificationv1alpha1.SlackNotificationRule
	for _, rule := range ruleList.Items {
		// Check TargetResource Kind
		// Assuming we pass the Kind string of targetObj or verify it outside.
		// Here we just check Selector.

		// If rule specifies namespace, it must match? Usually Rules are cluster-wide or namespaced?
		// CRD scope is Namespaced usually. So we filtered list by namespace?
		// If List(ctx, &ruleList) is called without params, and Rule is namespaced, it returns all in namespace (if cached client used with namespaced cache) or we need to check.
		// Let's assume Rules are in the same namespace as the Target.
		if rule.Namespace != targetObj.GetNamespace() {
			continue
		}

		selector, err := metav1.LabelSelectorAsSelector(&rule.Spec.LabelSelector)
		if err != nil {
			logger.Error(err, "invalid label selector in rule", "rule", rule.Name)
			continue
		}
		if selector.Matches(labels.Set(targetObj.GetLabels())) {
			matchedRules = append(matchedRules, rule)
		}
	}

	if len(matchedRules) == 0 {
		return nil
	}

	// For each matched rule, check if notification is needed
	for _, rule := range matchedRules {
		// Check TargetResource type matches
		// This should be passed or checked.
		// For now we rely on the Reconciler to only call for correct types or we check Kind here if available.

		for _, note := range rule.Spec.Notifications {
			if strings.EqualFold(note.Status, status) {
				if err := n.ResolveAndSend(ctx, rule, note, triggerObj); err != nil {
					logger.Error(err, "failed to send notification", "rule", rule.Name, "status", status)
				}
			}
		}
	}

	return nil
}

func (n *Notifier) ResolveAndSend(ctx context.Context, rule notificationv1alpha1.SlackNotificationRule, note notificationv1alpha1.NotificationRule, data any) error {
	var config notificationv1alpha1.SlackConfig
	if err := n.Client.Get(ctx, types.NamespacedName{Name: rule.Spec.SlackConfigRef.Name, Namespace: rule.Namespace}, &config); err != nil {
		return fmt.Errorf("failed to get slack config: %w", err)
	}

	var webhookURL string
	var token string

	if config.Spec.AuthType == "Webhook" {
		if config.Spec.WebhookURLSecretRef != nil {
			val, err := n.getSecretValue(ctx, config.Namespace, config.Spec.WebhookURLSecretRef)
			if err != nil {
				return err
			}
			webhookURL = val
		}
	} else if config.Spec.AuthType == "Token" {
		if config.Spec.TokenSecretRef != nil {
			val, err := n.getSecretValue(ctx, config.Namespace, config.Spec.TokenSecretRef)
			if err != nil {
				return err
			}
			token = val
		}
	}

	channel := config.Spec.Channel
	if note.Channel != "" {
		channel = note.Channel
	}

	// Convert to Unstructured map for template
	unstructuredData, err := runtime.DefaultUnstructuredConverter.ToUnstructured(data)
	if err != nil {
		return fmt.Errorf("failed to convert object to unstructured: %w", err)
	}

	// Determine Color
	color := note.Color
	if color == "" {
		// Default colors based on Status
		switch strings.ToLower(note.Status) {
		case "succeeded", "running":
			color = "good" // Green
		case "failed", "error":
			color = "danger" // Red
		default:
			color = "warning" // Orange
		}
	}

	return n.SlackClient.Send(ctx, webhookURL, token, channel, note.Title, note.Message, color, unstructuredData)
}

func (n *Notifier) getSecretValue(ctx context.Context, namespace string, ref *corev1.SecretKeySelector) (string, error) {
	var secret corev1.Secret
	if err := n.Client.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: namespace}, &secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", ref.Name, err)
	}
	val, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret %s", ref.Key, ref.Name)
	}
	return string(val), nil
}

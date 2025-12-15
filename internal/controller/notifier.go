package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	goslack "github.com/slack-go/slack"
	batchv1 "k8s.io/api/batch/v1"

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
// Notify checks rules and sends notifications.
// triggerObj: The object that triggered the event (e.g., Job, Workflow)
// targetObj: The object that rules target (e.g., CronJob, CronWorkflow)
func (n *Notifier) Notify(ctx context.Context, triggerObj client.Object, targetObj client.Object, status string) error {
	logger := log.FromContext(ctx)

	// List Rules in the target object's namespace
	var rules notificationv1alpha1.SlackNotificationRuleList
	if err := n.Client.List(ctx, &rules, client.InNamespace(targetObj.GetNamespace())); err != nil {
		return fmt.Errorf("failed to list rules: %w", err)
	}

	for _, rule := range rules.Items {
		// Check Target Resource
		targetRes := rule.Spec.TargetResource
		if targetRes == "CronJob" {
			if _, ok := targetObj.(*batchv1.CronJob); !ok {
				continue
			}
		} else if targetRes == "CronWorkflow" {
			if _, ok := targetObj.(*argov1alpha1.CronWorkflow); !ok {
				continue
			}
		}

		// Check Labels
		selector, err := metav1.LabelSelectorAsSelector(&rule.Spec.LabelSelector)
		if err != nil {
			logger.Error(err, "Invalid label selector", "rule", rule.Name)
			continue
		}
		if !selector.Matches(labels.Set(targetObj.GetLabels())) {
			continue
		}

		// Check Notification Config
		for _, note := range rule.Spec.Notifications {
			if strings.EqualFold(note.Status, status) {
				if err := n.ResolveAndSend(ctx, triggerObj, targetObj, rule, note); err != nil {
					logger.Error(err, "Failed to send notification", "rule", rule.Name)
				}
			}
		}
	}
	return nil
}

func (n *Notifier) ResolveAndSend(ctx context.Context, triggerObj client.Object, targetObj client.Object, rule notificationv1alpha1.SlackNotificationRule, note notificationv1alpha1.NotificationRule) error {
	// Get SlackConfig
	var config notificationv1alpha1.SlackConfig
	// SlackConfigRef is a LocalObjectReference, so it must be in the same namespace as the Rule
	ns := rule.Namespace

	if err := n.Client.Get(ctx, types.NamespacedName{Name: rule.Spec.SlackConfigRef.Name, Namespace: ns}, &config); err != nil {
		return fmt.Errorf("failed to get SlackConfig: %w", err)
	}

	// Resolve Credentials
	webhookURL := ""
	token := ""
	if config.Spec.AuthType == "Webhook" {
		if config.Spec.WebhookURLSecretRef != nil {
			val, err := n.getSecretValue(ctx, ns, config.Spec.WebhookURLSecretRef)
			if err != nil {
				return fmt.Errorf("failed to get webhook secret: %w", err)
			}
			webhookURL = val
		}
	} else if config.Spec.AuthType == "Token" {
		if config.Spec.TokenSecretRef != nil {
			val, err := n.getSecretValue(ctx, ns, config.Spec.TokenSecretRef)
			if err != nil {
				return fmt.Errorf("failed to get token secret: %w", err)
			}
			token = val
		}
	}

	channel := config.Spec.Channel
	if note.Channel != "" {
		channel = note.Channel
	}

	// Convert to Unstructured map for template
	unstructuredData, err := runtime.DefaultUnstructuredConverter.ToUnstructured(triggerObj)
	if err != nil {
		return fmt.Errorf("failed to convert object to unstructured: %w", err)
	}

	// Determine Color
	color := "warning"
	switch strings.ToLower(note.Status) {
	case "succeeded", "running":
		color = "good" // Green
	case "failed", "error":
		color = "danger" // Red
	}

	fields := n.buildFields(triggerObj, targetObj, note.Status)
	return n.SlackClient.Send(ctx, webhookURL, token, channel, note.Title, color, fields, unstructuredData)
}

func (n *Notifier) buildFields(triggerObj client.Object, targetObj client.Object, status string) []goslack.AttachmentField {
	namespace := targetObj.GetNamespace()
	ownerName := targetObj.GetName()
	ownerKind := targetObj.GetObjectKind().GroupVersionKind().Kind

	var duration string
	var reason string
	var message string

	// Extract details based on Trigger Object Type
	if job, ok := triggerObj.(*batchv1.Job); ok {
		if job.Status.StartTime != nil {
			endTime := metav1.Now()
			if job.Status.CompletionTime != nil {
				endTime = *job.Status.CompletionTime
			} else {
				for _, cond := range job.Status.Conditions {
					if (cond.Type == batchv1.JobFailed || cond.Type == batchv1.JobComplete) && cond.Status == corev1.ConditionTrue {
						endTime = cond.LastTransitionTime
						break
					}
				}
			}
			d := endTime.Time.Sub(job.Status.StartTime.Time)
			duration = d.Round(time.Second).String()
		}

		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobFailed && cond.Status == corev1.ConditionTrue {
				reason = cond.Reason
				message = cond.Message
				break
			}
		}
	} else if wf, ok := triggerObj.(*argov1alpha1.Workflow); ok {
		if !wf.Status.StartedAt.IsZero() {
			endTime := metav1.Now()
			if !wf.Status.FinishedAt.IsZero() {
				endTime = wf.Status.FinishedAt
			}
			d := endTime.Time.Sub(wf.Status.StartedAt.Time)
			duration = d.Round(time.Second).String()
		}
		message = wf.Status.Message
	}

	fields := []goslack.AttachmentField{
		{
			Title: "Namespace",
			Value: namespace,
			Short: true,
		},
		{
			Title: "Status",
			Value: status,
			Short: true,
		},
		{
			Title: ownerKind,
			Value: ownerName,
			Short: true,
		},
		{
			Title: "Duration",
			Value: duration,
			Short: true,
		},
	}

	if reason != "" {
		fields = append(fields, goslack.AttachmentField{
			Title: "Reason",
			Value: reason,
			Short: true,
		})
	}
	if message != "" {
		fields = append(fields, goslack.AttachmentField{
			Title: "Message",
			Value: message,
			Short: true,
		})
	}

	return fields
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

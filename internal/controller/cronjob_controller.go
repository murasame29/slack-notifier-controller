package controller

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/murasame29/slack-notifier-controller/internal/slack"
)

// CronJobReconciler reconciles a Job object owned by a CronJob
type CronJobReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Notifier *Notifier
}

// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch
// +kubebuilder:rbac:groups=notification.murasame29.com,resources=slacknotificationrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=notification.murasame29.com,resources=slackconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *CronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var job batchv1.Job
	if err := r.Get(ctx, req.NamespacedName, &job); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check owner reference
	var cronJobName string
	for _, ref := range job.OwnerReferences {
		if ref.Kind == "CronJob" {
			cronJobName = ref.Name
			break
		}
	}

	if cronJobName == "" {
		// Not owned by CronJob, ignore
		return ctrl.Result{}, nil
	}

	// Determine Status
	status := "Running"
	if job.Status.Succeeded > 0 {
		status = "Succeeded"
	} else if job.Status.Failed > 0 {
		status = "Failed"
	}
	// TODO: Handle more complex status or strictly look for conditions.
	// Simple check: active > 0 => Running?
	// But completed jobs stay.
	// We need to notify triggered by state change.
	// Since we are watching Job, we get called on update.
	// We should check if we already sent notification for this terminal state.

	// Check if already sent
	// NOTE: Ignoring annotation check for MVP to ensure notification logic works,
	// but strictly should check if annotation contains current status.
	// For "Running", it might spam. "Succeeded"/"Failed" is terminal.

	// Fetch Owner CronJob to pass as Target
	var cronJob batchv1.CronJob
	if err := r.Get(ctx, client.ObjectKey{Namespace: job.Namespace, Name: cronJobName}, &cronJob); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if err := r.Notifier.Notify(ctx, &job, &cronJob, status); err != nil {
		logger.Error(err, "Failed to notify")
		// Don't error out the reconciliation to avoid retry loops for notification failures unless critical
	}

	return ctrl.Result{}, nil
}

func (r *CronJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Notifier = &Notifier{
		Client:      mgr.GetClient(), // Changed from r.Client to mgr.GetClient()
		SlackClient: slack.NewClient(),
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.Job{}).
		Complete(r)
}

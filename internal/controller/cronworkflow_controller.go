package controller

import (
	"context"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/murasame29/slack-notifier-controller/internal/slack"
)

// CronWorkflowReconciler reconciles a Workflow object owned by a CronWorkflow
type CronWorkflowReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Notifier *Notifier
}

// +kubebuilder:rbac:groups=argoproj.io,resources=workflows,verbs=get;list;watch
// +kubebuilder:rbac:groups=argoproj.io,resources=cronworkflows,verbs=get;list;watch

func (r *CronWorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var wf argov1alpha1.Workflow
	if err := r.Get(ctx, req.NamespacedName, &wf); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check owner reference
	var cronWfName string
	for _, ref := range wf.OwnerReferences {
		if ref.Kind == "CronWorkflow" {
			cronWfName = ref.Name
			break
		}
	}

	if cronWfName == "" {
		return ctrl.Result{}, nil
	}

	// Determine Status
	status := string(wf.Status.Phase)
	// Argo phases: Running, Succeeded, Failed, Error, etc.

	// Fetch Owner CronWorkflow
	var cronWf argov1alpha1.CronWorkflow
	if err := r.Get(ctx, client.ObjectKey{Namespace: wf.Namespace, Name: cronWfName}, &cronWf); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if err := r.Notifier.Notify(ctx, &wf, &cronWf, status); err != nil {
		logger.Error(err, "Failed to notify")
	}

	return ctrl.Result{}, nil
}

func (r *CronWorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Notifier = &Notifier{
		Client:      mgr.GetClient(), // Initialize Notifier's Client with the manager's client
		SlackClient: slack.NewClient(),
	}
	// Note: You must register argov1alpha1 Scheme in main.go
	return ctrl.NewControllerManagedBy(mgr).
		For(&argov1alpha1.Workflow{}).
		Complete(r)
}

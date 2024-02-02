/*
Copyright 2023.

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

package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/nsf/jsondiff"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	ingressv1 "github.com/toro-ponz/dynamic-ingress-operator/api/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DEBUG = 1
)

// DynamicIngressReconciler reconciles a DynamicIngress object
type DynamicIngressReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ingress.toroponz.io,resources=dynamicingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ingress.toroponz.io,resources=dynamicingresses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ingress.toroponz.io,resources=dynamicingresses/finalizers,verbs=update
//+kubebuilder:rbac:groups=ingress.toroponz.io,resources=dynamicingressstates,verbs=get;list;watch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DynamicIngress object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *DynamicIngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var dynamicIngress ingressv1.DynamicIngress
	err := r.Get(ctx, req.NamespacedName, &dynamicIngress)
	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}

	if err != nil {
		logger.Error(err, "unable to get DynamicIngress", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	if !dynamicIngress.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	err = r.reconcileIngress(ctx, dynamicIngress)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

const (
	dynamicIngressStateField = ".spec.state"
)

// SetupWithManager sets up the controller with the Manager.
func (r *DynamicIngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(context.Background(), &ingressv1.DynamicIngress{}, dynamicIngressStateField, func(rawObj client.Object) []string {
		ingress := rawObj.(*ingressv1.DynamicIngress)
		return []string{ingress.Spec.State}
	})
	if err != nil {
		return err
	}

	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			old := e.ObjectOld.(*ingressv1.DynamicIngressState)
			new := e.ObjectNew.(*ingressv1.DynamicIngressState)
			if new.Status.LastUpdateTime == nil {
				return false
			}
			if old.Status.LastUpdateTime == nil {
				return true
			}
			return !old.Status.LastUpdateTime.Equal(new.Status.LastUpdateTime)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&ingressv1.DynamicIngress{}).
		Owns(&networkingv1.Ingress{}).
		Watches(
			&ingressv1.DynamicIngressState{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForDynamicIngressState),
			builder.WithPredicates(p),
		).
		Complete(r)
}

func (r *DynamicIngressReconciler) findObjectsForDynamicIngressState(ctx context.Context, state client.Object) []reconcile.Request {
	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("findObjectsForDynamicIngressState state=%s", state.GetName()))

	attachedDynamicIngresses := &ingressv1.DynamicIngressList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(dynamicIngressStateField, state.GetName()),
	}
	err := r.List(context.TODO(), attachedDynamicIngresses, listOps)
	if err != nil {
		logger.Error(err, fmt.Sprintf("findObjectsForDynamicIngressState state=%s", state.GetName()))
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(attachedDynamicIngresses.Items))
	for i, item := range attachedDynamicIngresses.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}

	return requests
}

func (r *DynamicIngressReconciler) reconcileIngress(ctx context.Context, dynamicIngress ingressv1.DynamicIngress) error {
	logger := log.FromContext(ctx)

	target := dynamicIngress.Spec.Target

	isActive, err := r.checkConditions(ctx, dynamicIngress)
	if err != nil {
		return err
	}

	if isActive {
		err = r.applyIngress(ctx, dynamicIngress, target, dynamicIngress.Spec.ActiveIngress)
		if err != nil {
			logger.Error(err, fmt.Sprintf("unable to apply active ingress (name=%s, namespace=%s)", target.Name, target.Name))
			return err
		}

		logger.Info(fmt.Sprintf("succeeded apply active ingress (name=%s, namespace=%s)", target.Name, target.Name))
	} else {
		err = r.applyIngress(ctx, dynamicIngress, target, dynamicIngress.Spec.PassiveIngress)
		if err != nil {
			logger.Error(err, fmt.Sprintf("unable to apply passive ingress (name=%s, namespace=%s)", target.Name, target.Name))
			return err
		}

		logger.Info(fmt.Sprintf("succeeded apply passive ingress (name=%s, namespace=%s)", target.Name, target.Name))
	}

	return nil
}

func (r *DynamicIngressReconciler) applyIngress(
	ctx context.Context,
	dynamicIngress ingressv1.DynamicIngress,
	target ingressv1.DynamicIngressTarget,
	ingressTemplate *ingressv1.DynamicIngressTemplate,
) error {
	logger := log.FromContext(ctx)

	ingress := &networkingv1.Ingress{}
	ingress.SetName(target.Name)
	ingress.SetNamespace(target.Namespace)

	if ingressTemplate != nil {
		logger.V(DEBUG).Info(fmt.Sprintf("start createOrUpdate ingress (name=%s, namespace=%s)", ingress.Name, ingress.Namespace))

		op, err := ctrl.CreateOrUpdate(ctx, r.Client, ingress, func() error {
			ingress.Spec = ingressTemplate.Template.Spec
			ingress.Annotations = ingressTemplate.Template.Metadata.Annotations
			ingress.Labels = ingressTemplate.Template.Metadata.Labels
			return ctrl.SetControllerReference(&dynamicIngress, ingress, r.Scheme)
		})

		if err != nil {
			return err
		}

		if op != controllerutil.OperationResultNone {
			logger.Info(fmt.Sprintf("ingress changes applied (name=%s, namespace=%s)", ingress.Name, ingress.Namespace))
			return nil
		}

		logger.V(DEBUG).Info(fmt.Sprintf("end createOrUpdate ingress (name=%s, namespace=%s)", ingress.Name, ingress.Namespace))
	} else {
		logger.V(DEBUG).Info(fmt.Sprintf("start delete ingress (name=%s, namespace=%s)", ingress.Name, ingress.Namespace))

		err := r.Get(ctx, client.ObjectKey{Namespace: ingress.Namespace, Name: ingress.Name}, ingress)
		if err != nil && apierrors.IsNotFound(err) {
			logger.V(DEBUG).Info(fmt.Sprintf("ingress not found (name=%s, namespace=%s)", ingress.Name, ingress.Namespace))
			return nil
		}
		if err != nil {
			return err
		}

		uid := ingress.GetUID()
		resourceVersion := ingress.GetResourceVersion()
		err = r.Delete(ctx, ingress, &client.DeleteOptions{
			Preconditions: &metav1.Preconditions{
				UID:             &uid,
				ResourceVersion: &resourceVersion,
			},
		})
		if err != nil {
			return err
		}

		logger.Info(fmt.Sprintf("ingress deleted (name=%s, namespace=%s)", ingress.Name, ingress.Namespace))
		logger.V(DEBUG).Info(fmt.Sprintf("end createOrUpdate ingress (name=%s, namespace=%s)", ingress.Name, ingress.Namespace))
	}

	return nil
}

func (r *DynamicIngressReconciler) checkConditions(ctx context.Context, dynamicIngress ingressv1.DynamicIngress) (bool, error) {
	logger := log.FromContext(ctx)

	var dynamicIngressState ingressv1.DynamicIngressState
	err := r.Get(ctx, client.ObjectKey{Name: dynamicIngress.Spec.State}, &dynamicIngressState)

	// TODO: refactoring
	if err != nil {
		logger.Error(err, "unable to get DynamicIngressState", "name", dynamicIngress.Spec.State)

		if dynamicIngress.Spec.ErrorPolicy == "active" {
			return true, nil
		} else if dynamicIngress.Spec.ErrorPolicy == "passive" {
			return false, nil
		} else if dynamicIngress.Spec.ErrorPolicy == "retain" {
			return false, err
		}

		return false, err
	}

	// TODO: refactoring
	if dynamicIngressState.Status.Response.Status != dynamicIngress.Spec.Expected.Status {
		err = fmt.Errorf(
			"[DynamicIngress] Failed status state. namespace=%s, name=%s, actual=%d, expected=%d",
			dynamicIngress.Namespace,
			dynamicIngress.Name,
			dynamicIngressState.Status.Response.Status,
			dynamicIngress.Spec.Expected.Status,
		)

		if dynamicIngress.Spec.ErrorPolicy == "active" {
			return true, nil
		} else if dynamicIngress.Spec.ErrorPolicy == "passive" {
			return false, nil
		} else if dynamicIngress.Spec.ErrorPolicy == "retain" {
			return false, err
		}

		return false, err
	}

	if dynamicIngress.Spec.Expected.CompareType == "json" {
		// json compare
		diffOpts := jsondiff.DefaultJSONOptions()
		res, _ := jsondiff.Compare([]byte(dynamicIngressState.Status.Response.Body), []byte(dynamicIngress.Spec.Expected.Body), &diffOpts)
		logger.Info(fmt.Sprintf("[DynamicIngress] JSON Diff %s", res))
		if res == jsondiff.FullMatch {
			return true, nil
		} else if res == jsondiff.NoMatch {
			return false, nil
		} else if res == jsondiff.SupersetMatch {
			if dynamicIngress.Spec.Expected.Policy == "contains" {
				return true, nil
			} else {
				return false, nil
			}
		}

		return false, fmt.Errorf("[DynamicIngress] JSON Compare error. %s", res)
	} else if dynamicIngress.Spec.Expected.CompareType == "plaintext" {
		// plain text compare
		if dynamicIngressState.Status.Response.Body == dynamicIngress.Spec.Expected.Body {
			return true, nil
		}

		if dynamicIngress.Spec.Expected.Policy == "contains" {
			return strings.Contains(dynamicIngressState.Status.Response.Body, dynamicIngress.Spec.Expected.Body), nil
		} else {
			return false, nil
		}
	}

	return false, fmt.Errorf("[DynamicIngress] invalid compare type %s", dynamicIngress.Spec.Expected.CompareType)
}

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
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ingressv1 "github.com/toro-ponz/dynamic-ingress-operator/api/v1"
)

// DynamicIngressStateReconciler reconciles a DynamicIngressState object
type DynamicIngressStateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ingress.toroponz.io,resources=dynamicingressstates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ingress.toroponz.io,resources=dynamicingressstates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ingress.toroponz.io,resources=dynamicingressstates/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DynamicIngressState object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *DynamicIngressStateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var dynamicIngressState ingressv1.DynamicIngressState
	err := r.Get(ctx, req.NamespacedName, &dynamicIngressState)
	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}

	if err != nil {
		logger.Error(err, "[DynamicIngressState] unable to get DynamicIngressState", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	if !dynamicIngressState.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	err = r.reconcileIngressState(ctx, dynamicIngressState)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DynamicIngressStateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ingressv1.DynamicIngressState{}).
		Complete(r)
}

func (r *DynamicIngressStateReconciler) reconcileIngressState(ctx context.Context, dynamicIngressState ingressv1.DynamicIngressState) error {
	logger := log.FromContext(ctx)

	if dynamicIngressState.Spec.FixedResponse != nil {
		dynamicIngressState.Status.Response = dynamicIngressState.Spec.FixedResponse
	}

	now := &v1.Time{Time: time.Now()}
	dynamicIngressState.Status.LastUpdateTime = now
	err := r.Status().Update(ctx, &dynamicIngressState)
	if err != nil {
		return err
	}

	logger.V(DEBUG).Info(fmt.Sprintf("[DynamicIngressState] reconcileIngressState name=%s, lastUpdateTime=%s", dynamicIngressState.Name, now))

	return nil
}
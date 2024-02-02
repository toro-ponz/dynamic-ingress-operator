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
	"io"
	"net/http"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

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
	logger.V(DEBUG).Info(fmt.Sprintf("[DynamicIngressState] Reconcile"))

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
	events := make(chan event.GenericEvent)
	source := source.Channel{
		Source:         events,
		DestBufferSize: 0,
	}

	interval, err := time.ParseDuration("10s")
	if err != nil {
		return err
	}

	err = mgr.Add(&ticker{
		events:   events,
		interval: interval,
	})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&ingressv1.DynamicIngressState{}).
		WatchesRawSource(&source, handler.EnqueueRequestsFromMapFunc(r.findList)).
		Complete(r)
}

func (r *DynamicIngressStateReconciler) findList(ctx context.Context, _ client.Object) []reconcile.Request {
	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("[DynamicIngressState] findList"))

	dynamicIngressstates := ingressv1.DynamicIngressStateList{}
	err := r.List(ctx, &dynamicIngressstates)
	if err != nil {
		return []reconcile.Request{}
	}

	var requests []reconcile.Request
	for _, dynamicIngressstate := range dynamicIngressstates.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: client.ObjectKey{Name: dynamicIngressstate.Name},
		})
	}

	return requests
}

func (r *DynamicIngressStateReconciler) reconcileIngressState(ctx context.Context, dynamicIngressState ingressv1.DynamicIngressState) error {
	logger := log.FromContext(ctx)

	if dynamicIngressState.Spec.FixedResponse != nil {
		if dynamicIngressState.Status.Response == dynamicIngressState.Spec.FixedResponse {
			logger.V(DEBUG).Info(fmt.Sprintf("[DynamicIngressState] skip update fixedResponse name=%s", dynamicIngressState.Name))
			return nil
		}
		dynamicIngressState.Status.Response = dynamicIngressState.Spec.FixedResponse
	} else if dynamicIngressState.Spec.Probe != nil {
		response, err := r.probe(ctx, *dynamicIngressState.Spec.Probe)
		if err != nil {
			logger.Error(err, fmt.Sprintf("[DynamicIngressState] Probe error name=%s", dynamicIngressState.Name))
			return err
		}
		dynamicIngressState.Status.Response = response
	} else {
		return fmt.Errorf("[DynamicIngressState] Need fixedResponse or probe name=%s", dynamicIngressState.Name)
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

func (r *DynamicIngressStateReconciler) probe(ctx context.Context, probe ingressv1.DynamicIngressStateProbe) (*ingressv1.DynamicIngressStateResponse, error) {
	logger := log.FromContext(ctx)
	logger.V(DEBUG).Info(fmt.Sprintf("[DynamicIngressState] Do Probe %v", probe))

	if probe.Type != "HTTP" {
		return nil, fmt.Errorf("[DynamicIngressState] Probe type is HTTP only")
	}

	client := &http.Client{}
	req, err := http.NewRequest(probe.Method, probe.Url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &ingressv1.DynamicIngressStateResponse{
		Status: resp.StatusCode,
		Body:   string(body),
	}, nil
}

type ticker struct {
	events chan event.GenericEvent

	interval time.Duration
}

func (t *ticker) Start(ctx context.Context) error {
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			t.events <- event.GenericEvent{}
		}
	}
}

func (t *ticker) NeedLeaderElection() bool {
	return true
}

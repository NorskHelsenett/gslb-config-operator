/*
Copyright 2026.

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

	"codeberg.org/miekg/dns"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/NorskHelsenett/gslb-config-operator/api/v1alpha1"
	lbv1alpha1 "github.com/NorskHelsenett/gslb-config-operator/api/v1alpha1"
	"github.com/NorskHelsenett/gslb-config-operator/internal/config"
	"github.com/NorskHelsenett/gslb-config-operator/internal/gslb"
	dnsClient "github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/records"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/zones"
)

// GSLBServiceReconciler reconciles a GSLBService object
type GSLBServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	DNS    dnsClient.Client
}

// targetKey identifies a target object referenced by a GSLBService.
type targetKey struct {
	kind      string
	namespace string
	name      string
}

var configZoneID string

// +kubebuilder:rbac:groups=lb.nhn.no,resources=gslbservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=lb.nhn.no,resources=gslbservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=lb.nhn.no,resources=gslbservices/finalizers,verbs=update
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes;tcproutes;gateways,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *GSLBServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	logger := logf.FromContext(ctx)
	logger.Info("reconciling GSLBService", "request", req.String())

	svc := &v1alpha1.GSLBService{}
	if err := r.Get(ctx, req.NamespacedName, svc); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Persist any status mutations accumulated during reconciliation on return.
	base := svc.DeepCopy()
	defer func() {
		if statusErr := r.patchStatus(ctx, base, svc); statusErr != nil {
			logger.Error(statusErr, "failed to patch GSLBService status")
			if err == nil {
				err = statusErr
				result = ctrl.Result{}
			}
		}
	}()

	reconcilers := []subReconciler{
		r.reconcileFinalizer,
		r.reconcileTarget,
		r.reconcileConfig,
	}
	if config.DNS().Gen() == config.G3 {
		reconcilers = append(reconcilers, r.reconcileG3Record, r.reconcileG3NSRecords)
	}

	rc := &reconcileContext{service: svc}
	for _, phase := range reconcilers {
		if result, err = phase(ctx, rc); err != nil || !result.IsZero() || rc.stop {
			return result, err
		}
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// ensureConfigZoneID lazily resolves and caches the GSLB config-zone ID.
func (r *GSLBServiceReconciler) ensureConfigZoneID() error {
	if configZoneID != "" {
		return nil
	}

	configZone, err := r.DNS.Zones().Read(
		zones.WithFQDN(config.DNS().Zone()),
		zones.WithInfrastructure(config.DNS().ZoneInfra()),
	)
	if err != nil {
		return err
	}

	configZoneID = configZone.ID
	return nil
}

func (r *GSLBServiceReconciler) deleteRecord(ctx context.Context, svc *v1alpha1.GSLBService) error {
	if svc.Status.Member == nil || svc.Status.Member.ID == "" {
		// Never programmed; nothing to delete.
		return nil
	}

	r.DNS.Zones().Read()

	if err := r.ensureConfigZoneID(); err != nil {
		return fmt.Errorf("fetch config-zone: %w", err)
	}

	recordRequests, err := r.DNS.Records().Read(
		records.WithZoneID(configZoneID),
		records.WithRecordName(fmt.Sprintf("%s.%s.", svc.Status.Member.ID, config.DNS().Zone())),
	)
	if err != nil {
		return fmt.Errorf("read config records: %w", err)
	}

	if len(recordRequests) == 0 {
		return nil
	}

	if err := r.DNS.Records().Delete(recordRequests[0].RecordID); err != nil {
		return fmt.Errorf("failed to delete TXT config record: %w", err)
	}

	if config.DNS().Gen() == config.G3 {
		return r.deleteNSRecords(ctx, svc)
	}

	return nil
}

// deleteNSRecords removes the memberOf delegation NS records this member added.
func (r *GSLBServiceReconciler) deleteNSRecords(ctx context.Context, svc *v1alpha1.GSLBService) error {
	nameServers := config.DNS().NameServers()
	if len(nameServers) == 0 || svc.Status.Member == nil || svc.Status.Member.MemberOf == "" {
		return nil
	}

	recordName := svc.Status.Member.MemberOf
	if !strings.HasSuffix(recordName, ".") {
		recordName += "."
	}

	wantNS := make(map[string]struct{}, len(nameServers))
	for _, ns := range nameServers {
		wantNS[ns] = struct{}{}
	}

	infrastructures := map[string]struct{}{}
	for _, view := range svc.Status.Member.Views {
		if infra, ok := gslb.ViewsToDNSInfrastructure[string(view)]; ok {
			infrastructures[infra] = struct{}{}
		}
	}

	for infra := range infrastructures {
		zone, err := r.DNS.Zones().Read(
			zones.WithFQDN(svc.Status.Member.MemberOf),
			zones.WithInfrastructure(infra),
		)
		if err != nil {
			return fmt.Errorf("resolve memberOf zone (%s): %w", infra, err)
		}

		existing, err := r.DNS.Records().Read(
			records.WithZoneID(zone.ID),
			records.WithRecordName(recordName),
		)
		if err != nil {
			return fmt.Errorf("read memberOf NS records (%s): %w", infra, err)
		}

		for _, req := range existing {
			ns, ok := req.RR.(*dns.NS)
			if !ok {
				continue
			}
			if _, ok := wantNS[ns.Ns]; !ok {
				continue
			}
			if err := r.DNS.Records().Delete(req.RecordID); err != nil {
				return fmt.Errorf("delete memberOf NS record %q (%s): %w", ns.Ns, infra, err)
			}
			logf.FromContext(ctx).Info("Deleted memberOf NS record", "nameserver", ns.Ns, "infrastructure", infra)
		}
	}

	return nil
}

// patchStatus writes status changes accumulated during reconciliation, skipping no-op patches.
func (r *GSLBServiceReconciler) patchStatus(ctx context.Context, base, current *v1alpha1.GSLBService) error {
	if equality.Semantic.DeepEqual(base.Status, current.Status) {
		return nil
	}
	return r.Status().Patch(ctx, current, client.MergeFrom(base))
}

// setCondition upserts a status condition observed at the current generation.
func setCondition(g *v1alpha1.GSLBService, condType string, status metav1.ConditionStatus, reason, message string) {
	apimeta.SetStatusCondition(&g.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: g.Generation,
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *GSLBServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&lbv1alpha1.GSLBService{}).
		Watches(&gatewayv1.HTTPRoute{}, handler.EnqueueRequestsFromMapFunc(r.gslbServicesForTargetFunc("HTTPRoute"))).
		Watches(&gatewayv1.TCPRoute{}, handler.EnqueueRequestsFromMapFunc(r.gslbServicesForTargetFunc("TCPRoute"))).
		Watches(&corev1.Service{}, handler.EnqueueRequestsFromMapFunc(r.gslbServicesForTargetFunc("Service"))).
		Watches(&gatewayv1.Gateway{}, handler.EnqueueRequestsFromMapFunc(r.gslbServicesForGateway)).
		Named("gslbservice").
		Complete(r)
}

// gslbServicesForTargetFunc returns a map func enqueuing GSLBServices whose
// targetRef of the given kind matches the changed object.
func (r *GSLBServiceReconciler) gslbServicesForTargetFunc(kind string) handler.MapFunc {
	return func(ctx context.Context, obj client.Object) []ctrl.Request {
		return r.requestsForTargets(ctx, map[targetKey]struct{}{
			{kind: kind, namespace: obj.GetNamespace(), name: obj.GetName()}: {},
		})
	}
}

// gslbServicesForGateway enqueues GSLBServices whose target route has the
// changed Gateway as a parent.
func (r *GSLBServiceReconciler) gslbServicesForGateway(ctx context.Context, obj client.Object) []ctrl.Request {
	gwNamespace, gwName := obj.GetNamespace(), obj.GetName()
	targets := map[targetKey]struct{}{}

	var httpRoutes gatewayv1.HTTPRouteList
	if err := r.List(ctx, &httpRoutes); err == nil {
		for i := range httpRoutes.Items {
			route := &httpRoutes.Items[i]
			if routeHasParentGateway(route.Namespace, route.Spec.ParentRefs, gwNamespace, gwName) {
				targets[targetKey{kind: "HTTPRoute", namespace: route.Namespace, name: route.Name}] = struct{}{}
			}
		}
	}

	var tcpRoutes gatewayv1.TCPRouteList
	if err := r.List(ctx, &tcpRoutes); err == nil {
		for i := range tcpRoutes.Items {
			route := &tcpRoutes.Items[i]
			if routeHasParentGateway(route.Namespace, route.Spec.ParentRefs, gwNamespace, gwName) {
				targets[targetKey{kind: "TCPRoute", namespace: route.Namespace, name: route.Name}] = struct{}{}
			}
		}
	}

	return r.requestsForTargets(ctx, targets)
}

// requestsForTargets lists GSLBServices and enqueues those whose resolved
// targetRef matches one of the given targets.
func (r *GSLBServiceReconciler) requestsForTargets(ctx context.Context, targets map[targetKey]struct{}) []ctrl.Request {
	if len(targets) == 0 {
		return nil
	}

	var list lbv1alpha1.GSLBServiceList
	if err := r.List(ctx, &list); err != nil {
		logf.FromContext(ctx).Error(err, "Failed to list GSLBServices for watch mapping")
		return nil
	}

	var reqs []ctrl.Request
	for i := range list.Items {
		gslb := &list.Items[i]
		ref := gslb.Spec.Member.TargetRef
		refNamespace := gslb.Namespace
		if ref.Namespace != nil {
			refNamespace = *ref.Namespace
		}
		if _, ok := targets[targetKey{kind: ref.Kind, namespace: refNamespace, name: ref.Name}]; !ok {
			continue
		}
		reqs = append(reqs, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: gslb.Namespace, Name: gslb.Name}})
	}
	return reqs
}

// routeHasParentGateway reports whether parents references the given Gateway.
func routeHasParentGateway(routeNamespace string, parents []gatewayv1.ParentReference, gwNamespace, gwName string) bool {
	for _, p := range parents {
		if p.Kind != nil && string(*p.Kind) != "Gateway" {
			continue
		}
		if string(p.Name) != gwName {
			continue
		}
		namespace := routeNamespace
		if p.Namespace != nil {
			namespace = string(*p.Namespace)
		}
		if namespace == gwNamespace {
			return true
		}
	}
	return false
}

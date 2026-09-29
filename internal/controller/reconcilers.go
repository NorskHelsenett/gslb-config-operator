package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"codeberg.org/miekg/dns"
	"github.com/NorskHelsenett/gslb-config-operator/api/v1alpha1"
	"github.com/NorskHelsenett/gslb-config-operator/internal/config"
	"github.com/NorskHelsenett/gslb-config-operator/internal/gslb"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/records"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/zones"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/gslb/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// reconcileContext carries state threaded through the reconcile phases.
type reconcileContext struct {
	service *v1alpha1.GSLBService
	member  *gslb.Member
	config  *models.GSLBConfig
	stop    bool // set by a phase to end reconciliation without requeueing
}

// subReconciler performs one phase of GSLBService reconciliation.
type subReconciler func(ctx context.Context, rc *reconcileContext) (ctrl.Result, error)

func (r *GSLBServiceReconciler) reconcileFinalizer(ctx context.Context, rc *reconcileContext) (ctrl.Result, error) {
	svc := rc.service

	if !svc.DeletionTimestamp.IsZero() {
		rc.stop = true
		if !controllerutil.ContainsFinalizer(svc, gslbServiceFinalizer) {
			return ctrl.Result{}, nil
		}

		if err := r.deleteRecord(ctx, svc); err != nil {
			logf.FromContext(ctx).Error(err, "failed to delete GSLB config record")
			return ctrl.Result{RequeueAfter: requeueAfter}, nil
		}
		controllerutil.RemoveFinalizer(svc, gslbServiceFinalizer)
		return ctrl.Result{}, r.Update(ctx, svc)
	}

	if controllerutil.AddFinalizer(svc, gslbServiceFinalizer) {
		return ctrl.Result{}, r.Update(ctx, svc)
	}

	return ctrl.Result{}, nil
}

func (r *GSLBServiceReconciler) reconcileTarget(ctx context.Context, rc *reconcileContext) (ctrl.Result, error) {
	svc := rc.service
	target, err := gslb.For(r.Client, svc)
	if err != nil {
		setCondition(svc, conditionAccepted, metav1.ConditionFalse, "UnsupportedTarget", err.Error())
		rc.stop = true
		return ctrl.Result{}, nil
	}

	member, err := target.Resolve(ctx)
	if err != nil {
		setCondition(svc, conditionAccepted, metav1.ConditionFalse, "TargetResolutionFailed", err.Error())
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}
	rc.member = member
	setCondition(svc, conditionAccepted, metav1.ConditionTrue, "Resolved", "Succesfully resolved target member")

	return ctrl.Result{}, nil
}

func (r *GSLBServiceReconciler) reconcileConfig(_ context.Context, rc *reconcileContext) (ctrl.Result, error) {
	svc := rc.service
	if svc.Status.Member == nil {
		svc.Status.Member = &v1alpha1.GSLBMemberStatus{}
	}

	if svc.Status.Member.ID == "" {
		svc.Status.Member.ID = gslb.ServiceID(svc) + "-" + config.Server().Cluster()
	}

	cfg, err := gslb.BuildConfig(r.Client, svc, rc.member)
	if err != nil {
		setCondition(svc, conditionAccepted, metav1.ConditionFalse, "GSLBConfigGenerationFailed", err.Error())
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	rc.config = cfg
	rc.config.ServiceID = svc.Status.Member.ID
	svc.Status.Member.MemberOf = rc.config.MemberOf

	svc.Status.Member.Views = make([]v1alpha1.View, len(rc.config.Views))
	for i, view := range rc.config.Views {
		svc.Status.Member.Views[i] = v1alpha1.View(view)
	}

	return ctrl.Result{}, nil
}

func (r *GSLBServiceReconciler) reconcileG3Record(ctx context.Context, rc *reconcileContext) (ctrl.Result, error) {
	svc := rc.service

	if err := r.ensureConfigZoneID(); err != nil {
		logf.FromContext(ctx).Error(err, "unable to fetch GSLB DNS config-zone")
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	recordRequests, err := r.DNS.Records().Read(
		records.WithZoneID(configZoneID),
		records.WithRecordName(fmt.Sprintf("%s.%s.", rc.config.ServiceID, config.DNS().Zone())),
	)
	if err != nil {
		setCondition(svc, conditionProgrammed, metav1.ConditionFalse, "RecordLookupFailed", err.Error())
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	rawConfig, err := json.Marshal(rc.config)
	if err != nil {
		setCondition(svc, conditionProgrammed, metav1.ConditionFalse, "ConfigMarshalFailed", err.Error())
		return ctrl.Result{}, fmt.Errorf("failed to marshal generated GSLB config: %w", err)
	}

	desired := &dns.TXT{
		Hdr: dns.Header{Name: fmt.Sprintf("%s.%s.", rc.config.ServiceID, config.DNS().Zone())},
		Txt: []string{string(rawConfig)},
	}

	// Create when the record is absent.
	if len(recordRequests) == 0 {
		if err := r.DNS.Records().Create(records.RecordRequest{RR: desired}); err != nil {
			logf.FromContext(ctx).Error(err, "failed to create GSLB config txt record")
			setCondition(svc, conditionProgrammed, metav1.ConditionFalse, "RecordCreateFailed", err.Error())
			return ctrl.Result{RequeueAfter: requeueAfter}, nil
		}
		logf.FromContext(ctx).Info("created GSLB config txt record")
		setCondition(svc, conditionProgrammed, metav1.ConditionTrue, "RecordCreated", "GSLB config record created")
		return ctrl.Result{}, nil
	}

	// Otherwise reconcile drift: update only when the stored config differs.
	existing := recordRequests[0]
	inSync, err := recordMatchesConfig(existing.RR, rc.config)
	if err != nil {
		setCondition(svc, conditionProgrammed, metav1.ConditionFalse, "RecordDecodeFailed", err.Error())
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}
	if inSync {
		setCondition(svc, conditionProgrammed, metav1.ConditionTrue, "RecordUpToDate", "GSLB config record up to date")
		return ctrl.Result{}, nil
	}

	if err := r.DNS.Records().Update(existing.RecordID, records.RecordRequest{RR: desired, RecordID: existing.RecordID}); err != nil {
		setCondition(svc, conditionProgrammed, metav1.ConditionFalse, "RecordUpdateFailed", err.Error())
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}
	setCondition(svc, conditionProgrammed, metav1.ConditionTrue, "RecordUpdated", "GSLB config record updated")
	return ctrl.Result{}, nil
}

func (r *GSLBServiceReconciler) reconcileG3NSRecords(ctx context.Context, rc *reconcileContext) (ctrl.Result, error) {
	svc := rc.service
	nameServers := config.DNS().NameServers()
	if len(nameServers) == 0 {
		return ctrl.Result{}, nil
	}

	recordName := rc.config.MemberOf
	if !strings.HasSuffix(recordName, ".") {
		recordName += "."
	}

	// NS records delegate the memberOf name, so they live in the zone hosting
	// that name for each infrastructure the member is published to.
	infrastructures := map[string]struct{}{}
	for _, view := range rc.config.Views {
		if infra, ok := gslb.ViewsToDNSInfrastructure[view]; ok {
			infrastructures[infra] = struct{}{}
		}
	}

	for infra := range infrastructures {
		zone, err := r.DNS.Zones().Read(
			zones.WithFQDN(rc.config.MemberOf),
			zones.WithInfrastructure(infra),
		)
		if err != nil {
			logf.FromContext(ctx).Error(err, "unable to resolve memberOf zone", "infrastructure", infra)
			setCondition(svc, conditionProgrammed, metav1.ConditionFalse, "NSZoneLookupFailed", err.Error())
			return ctrl.Result{RequeueAfter: requeueAfter}, nil
		}

		existing, err := r.DNS.Records().Read(
			records.WithZoneID(zone.ID),
			records.WithRecordName(recordName),
		)
		if err != nil {
			setCondition(svc, conditionProgrammed, metav1.ConditionFalse, "NSRecordLookupFailed", err.Error())
			return ctrl.Result{RequeueAfter: requeueAfter}, nil
		}

		haveNS := make(map[string]struct{}, len(existing))
		for _, req := range existing {
			if ns, ok := req.RR.(*dns.NS); ok {
				haveNS[ns.Ns] = struct{}{}
			}
		}

		for _, ns := range nameServers {
			if _, ok := haveNS[ns]; ok {
				continue
			}
			desired := &dns.NS{
				Hdr: dns.Header{Name: recordName},
				Ns:  ns,
			}
			if err := r.DNS.Records().Create(records.RecordRequest{RR: desired, Infrastructure: infra}); err != nil {
				logf.FromContext(ctx).Error(err, "failed to create memberOf NS record", "nameserver", ns, "infrastructure", infra)
				setCondition(svc, conditionProgrammed, metav1.ConditionFalse, "NSRecordCreateFailed", err.Error())
				return ctrl.Result{RequeueAfter: requeueAfter}, nil
			}
			logf.FromContext(ctx).Info("Created memberOf NS record", "nameserver", ns, "infrastructure", infra)
		}
	}

	return ctrl.Result{}, nil
}

// recordMatchesConfig reports whether the stored TXT record already encodes want.
func recordMatchesConfig(rr dns.RR, want *models.GSLBConfig) (bool, error) {
	txt, ok := rr.(*dns.TXT)
	if !ok {
		return false, fmt.Errorf("unexpected record type %T, want *dns.TXT", rr)
	}
	rawData := strings.Join(txt.Txt, "")
	var stored models.GSLBConfig
	if err := json.NewDecoder(strings.NewReader(rawData)).Decode(&stored); err != nil {
		return false, fmt.Errorf("decode stored GSLB config: %w", err)
	}

	return reflect.DeepEqual(stored, *want), nil
}

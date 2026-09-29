package gslb

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/NorskHelsenett/gslb-config-operator/api/v1alpha1"
	"github.com/NorskHelsenett/gslb-config-operator/internal/config"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/gslb/models"
)

var vitiZoneToViews = map[string][]string{
	"inet":         {"inet", "hnet"},
	"hnet-public":  {"inet", "hnet"},
	"hnet-private": {"hnet"},
}

var ViewsToDNSInfrastructure = map[string]string{
	"inet": "external",
	"hnet": "internal",
}

var DNSInfrastructureToViews = map[string]string{
	"internett": "inet",
	"helsenett": "hnet",
}

var ingressClassNamesToViews = map[string][]string{
	"avi-ingress-class-datacenter":          {"hnet"},
	"avi-ingress-class-helsenett":           {"hnet"},
	"avi-ingress-class-dualstack-internett": {"inet"},
	"avi-ingress-class-internett":           {"inet"},
	"avi-ingress-class-ipv6-internett":      {"inet"},
}

// For returns the Target implementation matching the GSLBService member kind.
func For(c client.Client, gslb *v1alpha1.GSLBService) (Target, error) {
	ref := gslb.Spec.Member.TargetRef
	namespace := gslb.Namespace
	if ref.Namespace != nil {
		namespace = *ref.Namespace
	}

	b := base{
		client: c,
		key:    types.NamespacedName{Namespace: namespace, Name: ref.Name},
		checks: gslb.Spec.Member.HealthChecks,
	}

	switch ref.Kind {
	case KindHTTPRoute:
		return &httpRouteTarget{base: b}, nil
	case KindTCPRoute:
		return &tcpRouteTarget{base: b}, nil
	case KindService:
		return &serviceTarget{base: b}, nil
	default:
		return nil, fmt.Errorf("unsupported target kind %q", ref.Kind)
	}
}

// BuildConfig assembles the GSLBConfig TXT payload from the GSLBService spec and
// the resolved member details.
func BuildConfig(gslb *v1alpha1.GSLBService, member *Member) *models.GSLBConfig {
	cfg := &models.GSLBConfig{
		MemberOf:   member.MemberOf,
		Address:    member.Address,
		Port:       member.Port,
		Path:       member.Path,
		CheckType:  member.CheckType,
		Datacenter: config.Server().Datacenter(),
		Priority:   int(gslb.Spec.Global.Priority),
		Views:      member.Views,
	}

	if gslb.Status.Member != nil {
		cfg.ServiceID = gslb.Status.Member.ID
	}

	if hc := gslb.Spec.Member.HealthChecks; hc != nil {
		if hc.FailureThreshold != nil {
			cfg.FailureThreshold = hc.FailureThreshold
		}
		if hc.Lua != nil {
			script := models.LuaScript(*hc.Lua)
			cfg.Script = &script
		}
	}

	return cfg
}

var idEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

func ServiceID(gslb *v1alpha1.GSLBService) string {
	key := gslb.Namespace + "/" + gslb.Name
	sum := sha256.Sum256([]byte(key))
	return strings.ToLower(idEncoding.EncodeToString(sum[:8]))
}

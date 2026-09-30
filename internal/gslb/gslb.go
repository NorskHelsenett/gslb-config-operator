package gslb

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
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
	case KindIngress:
		return &ingressTarget{base: b}, nil
	default:
		return nil, fmt.Errorf("unsupported target kind %q", ref.Kind)
	}
}

// BuildConfig assembles the GSLBConfig TXT payload from the GSLBService spec and
// the resolved member details.
func BuildConfig(c client.Client, gslb *v1alpha1.GSLBService, member *Member) (*models.GSLBConfig, error) {
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
		} else if hc.LuaRef != nil {
			configMap := corev1.ConfigMap{}
			err := c.Get(context.Background(), client.ObjectKey{Name: hc.LuaRef.TargetRef}, &configMap)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve lua configmap reference: %w", err)
			}

			rawScript, ok := configMap.Data[hc.LuaRef.Key]
			if !ok {
				return nil, fmt.Errorf("failed to resolve lua script in configmap: key %s does not exist", hc.LuaRef.Key)
			}

			script := models.LuaScript(rawScript)
			cfg.Script = &script
		}
	}

	return cfg, nil
}

var idEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

func ServiceID(gslb *v1alpha1.GSLBService) string {
	key := gslb.Namespace + "/" + gslb.Name
	sum := sha256.Sum256([]byte(key))
	return strings.ToLower(idEncoding.EncodeToString(sum[:8]))
}

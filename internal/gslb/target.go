package gslb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strconv"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/NorskHelsenett/gslb-config-operator/api/v1alpha1"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/gslb/models"
)

// Supported GSLBService member target kinds.
const (
	KindHTTPRoute = "HTTPRoute"
	KindTCPRoute  = "TCPRoute"
	KindService   = "Service"
	KindIngress   = "Ingress"
)
const (
	viewsAnnotation    = "dns.nhn.no/override-infrastructure"
	vitiZoneAnnotation = "ipam.vitistack.io/zone"
)

// Member holds the kind-specific network details resolved from a target object.
type Member struct {
	MemberOf  string
	Address   models.Address
	Port      string
	Path      *string
	CheckType string
	Views     []string
}

// Target resolves the kind-specific member details of a GSLBService so the
// controller can build a GSLBConfig without knowing the concrete target kind.
type Target interface {
	// Resolve reads the referenced object and returns its member details.
	Resolve(ctx context.Context) (*Member, error)
	// MemberOf returns the global service identity this member belongs to.
	MemberOf(ctx context.Context) (string, error)
	// Address returns the advertised address(es) of the member's target.
	Address(ctx context.Context) (models.Address, error)
	// Port returns the advertised port of the member's target.
	Port(ctx context.Context) (string, error)
	// Views returns the DNS infrastructure views of the member's target
	Views(ctx context.Context) ([]string, error)
}

// base carries state shared by every Target implementation.
type base struct {
	client client.Client
	key    types.NamespacedName
	checks *v1alpha1.HealthChecks
}

// checkType returns the configured check kind, falling back to def.
func (b base) checkType(def string) string {
	if b.checks != nil && b.checks.Kind != "" {
		return b.checks.Kind
	}
	return def
}

// path returns the configured health-check path, if any.
func (b base) path() *string {
	if b.checks != nil {
		return b.checks.Path
	}
	return nil
}

// resolveParentAddress resolves the advertised address from the first parent
// Gateway referenced by a route. Shared by every route-kind Target.
func (b base) resolveParentAddress(ctx context.Context, routeNamespace string, parents []gatewayv1.ParentReference) (models.Address, error) {
	if len(parents) == 0 {
		return models.Address{}, errors.New("no parent-references set on route")
	}

	parentRef := parents[0]
	namespace := routeNamespace // a nil parentRef namespace defaults to the route's namespace
	if parentRef.Namespace != nil {
		namespace = string(*parentRef.Namespace)
	}

	parent := &gatewayv1.Gateway{}
	if err := b.client.Get(ctx, client.ObjectKey{Name: string(parentRef.Name), Namespace: namespace}, parent); err != nil {
		return models.Address{}, fmt.Errorf("unable to fetch parent gateway: %w", err)
	}

	return gatewayStatusAddress(parent.Status.Addresses)
}

// gatewayStatusAddress converts a Gateway's status addresses into a models.Address.
func gatewayStatusAddress(addresses []gatewayv1.GatewayStatusAddress) (models.Address, error) {
	switch len(addresses) {
	case 0:
		return models.Address{}, nil
	case 1:
		addr := models.Address{IPFamily: "SingleStack"}
		ip, err := netip.ParseAddr(addresses[0].Value)
		if err != nil {
			return models.Address{}, fmt.Errorf("unable to parse address: %w", err)
		}
		assignIP(&addr, ip)
		return addr, nil
	default:
		addr := models.Address{IPFamily: "DualStack"}
		for _, statusAddress := range addresses {
			ip, err := netip.ParseAddr(statusAddress.Value)
			if err != nil {
				return models.Address{}, fmt.Errorf("failed to parse IP address: %w", err)
			}
			assignIP(&addr, ip)
		}
		if addr.IPv4 == nil || addr.IPv6 == nil {
			return models.Address{}, errors.New("unable to set DualStack address")
		}
		return addr, nil
	}
}

// resolveParentPort resolves the advertised port from the first parent Gateway
// referenced by a route. Shared by every route-kind Target.
func (b base) resolveParentPort(ctx context.Context, routeNamespace string, parents []gatewayv1.ParentReference) (string, error) {
	if len(parents) == 0 {
		return "", errors.New("no parent-references set on route")
	}

	parentRef := parents[0]
	if parentRef.Port != nil {
		return strconv.Itoa(int(*parentRef.Port)), nil
	}

	namespace := routeNamespace // a nil parentRef namespace defaults to the route's namespace
	if parentRef.Namespace != nil {
		namespace = string(*parentRef.Namespace)
	}

	parent := &gatewayv1.Gateway{}
	if err := b.client.Get(ctx, client.ObjectKey{Name: string(parentRef.Name), Namespace: namespace}, parent); err != nil {
		return "", fmt.Errorf("unable to fetch parent gateway: %w", err)
	}

	return gatewayListenerPort(parent.Spec.Listeners, parentRef.SectionName)
}

// gatewayListenerPort returns the port of the Gateway listener named by
// sectionName, or the first listener's port when sectionName is nil.
func gatewayListenerPort(listeners []gatewayv1.Listener, sectionName *gatewayv1.SectionName) (string, error) {
	if len(listeners) == 0 {
		return "", errors.New("no listeners set on parent gateway")
	}
	if sectionName == nil {
		return strconv.Itoa(int(listeners[0].Port)), nil
	}
	for _, listener := range listeners {
		if listener.Name == *sectionName {
			return strconv.Itoa(int(listener.Port)), nil
		}
	}
	return "", fmt.Errorf("no listener %q found on parent gateway", *sectionName)
}

func (b base) resolveParentViews(ctx context.Context, routeNamespace, fqdn string, parents []gatewayv1.ParentReference) ([]string, error) {
	if len(parents) == 0 {
		return nil, fmt.Errorf("no parent-reference set on route")
	}

	parentRef := parents[0]
	namespace := routeNamespace
	if parentRef.Namespace != nil {
		namespace = string(*parentRef.Namespace)
	}

	parent := gatewayv1.Gateway{}
	if err := b.client.Get(ctx, client.ObjectKey{Name: string(parentRef.Name), Namespace: namespace}, &parent); err != nil {
		return nil, fmt.Errorf("unable to fetch parent gateway: %w", err)
	}
	return gatewayViews(parent, fqdn)
}

func gatewayViews(gw gatewayv1.Gateway, fqdn string) ([]string, error) {
	overrideInfrastructure, ok := gw.Annotations[viewsAnnotation]
	if ok {
		infrastructure := map[string][]string{}
		if err := json.Unmarshal([]byte(overrideInfrastructure), &infrastructure); err != nil {
			return nil, fmt.Errorf("failed to parse infrastructure annotation: %w", err)
		}

		infrastructures, ok := infrastructure[fqdn]
		if ok {
			views := make([]string, 0)
			for _, infra := range infrastructures {
				view, ok := DNSInfrastructureToViews[infra]
				if !ok {
					continue
				}
				views = append(views, view)
			}
		}
	}

	zone, ok := gw.Spec.Infrastructure.Annotations[vitiZoneAnnotation]
	if !ok {
		return nil, fmt.Errorf("missing %s annotation for gateway", vitiZoneAnnotation)
	}

	views, ok := vitiZoneToViews[string(zone)]
	if !ok {
		return nil, fmt.Errorf("invalid %s annotation value %s", vitiZoneAnnotation, zone)
	}

	return views, nil
}

func assignIP(addr *models.Address, ip netip.Addr) {
	if ip.Is4() {
		addr.IPv4 = &ip
	}
	if ip.Is6() {
		addr.IPv6 = &ip
	}
}

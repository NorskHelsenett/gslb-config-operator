package gslb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"slices"

	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/gslb/models"
	networkingv1 "k8s.io/api/networking/v1"
)

type ingressTarget struct {
	base
	route *networkingv1.Ingress
}

func (t *ingressTarget) ingress(ctx context.Context) (*networkingv1.Ingress, error) {
	if t.route != nil {
		return t.route, nil
	}

	var route networkingv1.Ingress
	if err := t.client.Get(ctx, t.key, &route); err != nil {
		return nil, fmt.Errorf("read Ingress %s: %w", t.key, err)
	}

	t.route = &route
	return t.route, nil
}

func (t *ingressTarget) Resolve(ctx context.Context) (*Member, error) {
	_, err := t.ingress(ctx)
	if err != nil {
		return nil, err
	}

	memberOf, err := t.MemberOf(ctx)
	if err != nil {
		return nil, err
	}
	address, err := t.Address(ctx)
	if err != nil {
		return nil, err
	}
	views, err := t.Views(ctx)
	if err != nil {
		return nil, err
	}
	port, err := t.Port(ctx)
	if err != nil {
		return nil, err
	}

	return &Member{
		MemberOf:  memberOf,
		Address:   address,
		Port:      port,
		CheckType: t.checkType("HTTPS"),
		Path:      t.path(),
		Views:     views,
	}, nil
}

func (t *ingressTarget) MemberOf(ctx context.Context) (string, error) {
	route, err := t.ingress(ctx)
	if err != nil {
		return "", err
	}

	if len(route.Status.LoadBalancer.Ingress) > 0 {
		return route.Status.LoadBalancer.Ingress[0].Hostname, nil
	} else if len(route.Spec.Rules) > 0 {
		return route.Spec.Rules[0].Host, nil
	}

	return "", errors.New("failed to resolve memberOf")
}

func (t *ingressTarget) Address(ctx context.Context) (models.Address, error) {
	route, err := t.ingress(ctx)
	if err != nil {
		return models.Address{}, err
	}

	return ingressLoadBalancerAddress(route.Status.LoadBalancer.Ingress)
}

func (t *ingressTarget) Port(ctx context.Context) (string, error) {
	route, err := t.ingress(ctx)
	if err != nil {
		return "", err
	}

	var host string
	if len(route.Spec.Rules) > 0 {
		host = route.Spec.Rules[0].Host
	}

	// External port depends on whether TLS terminates at the ingress controller, not the backend service port.
	for _, tls := range route.Spec.TLS {
		if len(tls.Hosts) == 0 || slices.Contains(tls.Hosts, host) {
			return "443", nil
		}
	}

	return "80", nil
}

func (t *ingressTarget) Views(ctx context.Context) ([]string, error) {
	route, err := t.ingress(ctx)
	if err != nil {
		return nil, err
	}

	overrideInfrastructure, ok := route.Annotations[viewsAnnotation]
	if ok {
		infrastructure := map[string][]string{}
		if err := json.Unmarshal([]byte(overrideInfrastructure), &infrastructure); err != nil {
			return nil, fmt.Errorf("failed to parse infrastructure annotation")
		}

		return infrastructure["infrastructure"], nil
	} else {
		if route.Spec.IngressClassName == nil {
			return nil, errors.New("no classname for ingress")
		}

		views, ok := ingressClassNamesToViews[*route.Spec.IngressClassName]
		if !ok {
			return nil, fmt.Errorf("un-supported ingress class name %s", *route.Spec.IngressClassName)
		}
		return views, nil
	}
}

// ingressLoadBalancerAddress converts an Ingress's LoadBalancer status entries into a models.Address.
func ingressLoadBalancerAddress(ingresses []networkingv1.IngressLoadBalancerIngress) (models.Address, error) {
	switch len(ingresses) {
	case 0:
		return models.Address{}, errors.New("no addresses set on Ingress status")
	case 1:
		if ingresses[0].IP == "" {
			return models.Address{}, fmt.Errorf("no IP set on Ingress status address %q", ingresses[0].Hostname)
		}

		addr := models.Address{IPFamily: "SingleStack"}
		ip, err := netip.ParseAddr(ingresses[0].IP)
		if err != nil {
			return models.Address{}, fmt.Errorf("unable to parse address: %w", err)
		}

		assignIP(&addr, ip)
		return addr, nil
	default:
		addr := models.Address{IPFamily: "DualStack"}
		for _, ingress := range ingresses {
			if ingress.IP == "" {
				continue
			}

			ip, err := netip.ParseAddr(ingress.IP)
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

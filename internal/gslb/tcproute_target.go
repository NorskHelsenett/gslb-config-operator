package gslb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/gslb/models"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type tcpRouteTarget struct {
	base
	route *gatewayv1.TCPRoute
}

func (t *tcpRouteTarget) tcpRoute(ctx context.Context) (*gatewayv1.TCPRoute, error) {
	if t.route != nil {
		return t.route, nil
	}

	var route gatewayv1.TCPRoute
	if err := t.client.Get(ctx, t.key, &route); err != nil {
		return nil, fmt.Errorf("read TCPRoute %s: %w", t.key, err)
	}

	t.route = &route
	return t.route, nil
}

func (t *tcpRouteTarget) MemberOf(ctx context.Context) (string, error) {
	route, err := t.tcpRoute(ctx)
	if err != nil {
		return "", err
	}

	memberOf, ok := route.Annotations["dns.nhn.no/fqdn"]
	if !ok {
		return "", fmt.Errorf("unable to resolve memberOf from Service: dns.nhn.no/fqdn annotation not set")
	}

	return memberOf, nil
}

func (t *tcpRouteTarget) Address(ctx context.Context) (models.Address, error) {
	route, err := t.tcpRoute(ctx)
	if err != nil {
		return models.Address{}, err
	}
	return t.resolveParentAddress(ctx, route.Namespace, route.Spec.ParentRefs)
}

func (t *tcpRouteTarget) Port(ctx context.Context) (string, error) {
	route, err := t.tcpRoute(ctx)
	if err != nil {
		return "", err
	}
	return t.resolveParentPort(ctx, route.Namespace, route.Spec.ParentRefs)
}

func (t *tcpRouteTarget) Views(ctx context.Context) ([]string, error) {
	route, err := t.tcpRoute(ctx)
	if err != nil {
		return nil, err
	}

	overrideInfrastructure, ok := route.Annotations[viewsAnnotation]
	if !ok {
		memberFqdn, err := t.MemberOf(ctx)
		if err != nil {
			return nil, err
		}

		return t.resolveParentViews(ctx, route.Namespace, memberFqdn, route.Spec.ParentRefs)
	}

	infrastructure := map[string][]string{}
	if err := json.Unmarshal([]byte(overrideInfrastructure), &infrastructure); err != nil {
		return nil, fmt.Errorf("failed to parse infrastructure annotation")
	}

	return infrastructure["infrastructure"], nil
}

func (t *tcpRouteTarget) Resolve(ctx context.Context) (*Member, error) {
	if _, err := t.tcpRoute(ctx); err != nil {
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
	port, err := t.Port(ctx)
	if err != nil {
		return nil, err
	}
	views, err := t.Views(ctx)
	if err != nil {
		return nil, err
	}

	return &Member{
		MemberOf:  memberOf,
		Address:   address,
		Port:      port,
		CheckType: t.checkType("TCP-FULL"),
		Path:      t.path(),
		Views:     views,
	}, nil
}

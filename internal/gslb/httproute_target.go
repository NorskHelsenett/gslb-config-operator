package gslb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/gslb/models"
)

type httpRouteTarget struct {
	base
	route *gatewayv1.HTTPRoute
}

func (t *httpRouteTarget) httpRoute(ctx context.Context) (*gatewayv1.HTTPRoute, error) {
	if t.route != nil {
		return t.route, nil
	}
	var route gatewayv1.HTTPRoute
	if err := t.client.Get(ctx, t.key, &route); err != nil {
		return nil, fmt.Errorf("read HTTPRoute %s: %w", t.key, err)
	}
	t.route = &route
	return t.route, nil
}

func (t *httpRouteTarget) MemberOf(ctx context.Context) (string, error) {
	route, err := t.httpRoute(ctx)
	if err != nil {
		return "", err
	}

	if len(route.Spec.Hostnames) == 0 {
		return "", errors.New("hostnames not set on HTTPRoute")
	}

	return string(route.Spec.Hostnames[0]), nil
}

func (t *httpRouteTarget) Address(ctx context.Context) (models.Address, error) {
	route, err := t.httpRoute(ctx)
	if err != nil {
		return models.Address{}, err
	}
	return t.resolveParentAddress(ctx, route.Namespace, route.Spec.ParentRefs)
}

func (t *httpRouteTarget) Port(ctx context.Context) (string, error) {
	route, err := t.httpRoute(ctx)
	if err != nil {
		return "", err
	}
	return t.resolveParentPort(ctx, route.Namespace, route.Spec.ParentRefs)
}

func (t *httpRouteTarget) Views(ctx context.Context) ([]string, error) {
	route, err := t.httpRoute(ctx)
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

	infrastructures := map[string][]string{}
	if err := json.Unmarshal([]byte(overrideInfrastructure), &infrastructures); err != nil {
		return nil, fmt.Errorf("failed to parse infrastructure annotation")
	}

	views := make([]string, 0)
	for _, infra := range infrastructures["infrastructure"] {
		view, ok := DNSInfrastructureToViews[infra]
		if !ok {
			continue
		}
		views = append(views, view)
	}

	return views, nil
}

func (t *httpRouteTarget) Resolve(ctx context.Context) (*Member, error) {
	if _, err := t.httpRoute(ctx); err != nil {
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

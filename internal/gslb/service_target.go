package gslb

import (
	"context"
	"fmt"
	"net/netip"
	"strconv"

	corev1 "k8s.io/api/core/v1"

	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/gslb/models"
)

type serviceTarget struct {
	base
	svc *corev1.Service
}

func (t *serviceTarget) service(ctx context.Context) (*corev1.Service, error) {
	if t.svc != nil {
		return t.svc, nil
	}
	var svc corev1.Service
	if err := t.client.Get(ctx, t.key, &svc); err != nil {
		return nil, fmt.Errorf("read Service %s: %w", t.key, err)
	}
	t.svc = &svc
	return t.svc, nil
}

func (t *serviceTarget) MemberOf(ctx context.Context) (string, error) {
	svc, err := t.service(ctx)
	if err != nil {
		return "", err
	}

	memberOf, ok := svc.Annotations["dns.nhn.no/fqdn"]
	if !ok {
		return "", fmt.Errorf("unable to resolve memberOf from Service: dns.nhn.no/fqdn annotation not set")
	}

	return memberOf, nil
}

func (t *serviceTarget) Address(ctx context.Context) (models.Address, error) {
	svc, err := t.service(ctx)
	if err != nil {
		return models.Address{}, err
	}

	switch svc.Spec.Type {
	case "LoadBalancer":
		addr := models.Address{}
		if len(svc.Status.LoadBalancer.Ingress) == 1 {
			addr.IPFamily = "SingleStack"

			ip, err := netip.ParseAddr(string(svc.Status.LoadBalancer.Ingress[0].IP))
			if err != nil {
				return models.Address{}, fmt.Errorf("failed to parse ip adress: %w", err)
			}

			addr.IPv4 = &ip
		}

		return addr, nil
	default:
		return models.Address{}, fmt.Errorf("unsupported service type: %s for address resolution: Service must be of type LoadBalancer", svc.Spec.Type)
	}
}

func (t *serviceTarget) Port(ctx context.Context) (string, error) {
	svc, err := t.service(ctx)
	if err != nil {
		return "", err
	}

	if len(svc.Spec.Ports) == 0 {
		return "", nil
	}

	return strconv.Itoa(int(svc.Spec.Ports[0].Port)), nil
}

func (t *serviceTarget) Views(ctx context.Context) ([]string, error) {
	svc, err := t.service(ctx)
	if err != nil {
		return nil, err
	}

	zone, ok := svc.Annotations[vitiZoneAnnotation]
	if !ok {
		//TODO: how to resolve this when in tanzu cluster??
	}

	views, ok := vitiZoneToViews[zone]
	if !ok {
		return nil, fmt.Errorf("invalid %s annotation value %s", vitiZoneAnnotation, zone)
	}

	return views, nil
}

func (t *serviceTarget) Resolve(ctx context.Context) (*Member, error) {
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

	member := &Member{
		MemberOf:  memberOf,
		Address:   address,
		Port:      port,
		Path:      t.path(),
		CheckType: t.checkType("TCP-FULL"),
		Views:     views,
	}

	return member, nil
}

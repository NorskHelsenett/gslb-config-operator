package g3zones

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/g3/models"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/transport"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/zones"
)

type Client interface {
	Read(...zones.ReadOption) (zones.Zone, error)
}

type g3ZonesClient struct {
	transport transport.Transport
	path      string
}

func NewG3ZonesClient(transport transport.Transport) *g3ZonesClient {
	return &g3ZonesClient{
		transport: transport,
		path:      "/dns/zone",
	}
}

func (c *g3ZonesClient) Read(opts ...zones.ReadOption) (*zones.Zone, error) {
	options := &zones.ReadOptions{}
	for _, opt := range opts {
		opt(options)
	}

	params := url.Values{}
	if options.FQDN != nil {
		params.Set("fqdn", *options.FQDN)
	}
	if options.Infrastructure != nil {
		params.Set("infrastructure", *options.Infrastructure)
	}

	path := c.path
	if len(params) > 0 {
		path = c.path + "?" + params.Encode()
	}

	var g3Zone models.G3Zone
	if err := c.transport.GetJSON(context.Background(), path, &g3Zone); err != nil {
		return nil, fmt.Errorf("could not read zone: %w", err)
	}

	return &zones.Zone{
		ID:             strconv.Itoa(g3Zone.Id),
		Name:           g3Zone.Name,
		DefaultTTL:     strconv.Itoa(g3Zone.Default_Ttl),
		Infrastructure: g3Zone.Infrastructure,
	}, nil
}

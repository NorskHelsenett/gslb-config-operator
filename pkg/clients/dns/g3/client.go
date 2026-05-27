package g3

import (
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns"
	g3records "github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/g3/records"
	g3token "github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/g3/token"
	g3zones "github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/g3/zones"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/records"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/transport"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/zones"
)

type Client interface {
	dns.Client
	Token() g3token.Client
}

type g3Client struct {
	records records.Client
	zones   zones.Client
	token   g3token.Client
}

func NewClient(transport transport.Transport) Client {
	return &g3Client{
		records: g3records.NewG3RecordsAdapter(transport),
		zones:   g3zones.NewG3ZonesClient(transport),
		token:   g3token.NewG3TokenClient(transport),
	}
}

func (c *g3Client) Records() records.Client {
	return c.records
}

func (c *g3Client) Zones() zones.Client {
	return c.zones
}

func (c *g3Client) Token() g3token.Client {
	return c.token
}

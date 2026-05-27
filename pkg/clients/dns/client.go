package dns

import (
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/records"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/zones"
)

type Client interface {
	Records() records.Client
	Zones() zones.Client
}

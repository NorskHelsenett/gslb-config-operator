package initalize

import (
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Run(dnsClient dns.Client, k8sClient client.Client) error {
	return nil
}

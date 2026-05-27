package intercept

import (
	"net/http"

	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/transport/rest"
)

func WithAuth(header, token string) rest.Option {
	return rest.WithRoundTripper(
		NewAuthInterceptor(http.DefaultTransport, header, token),
	)
}

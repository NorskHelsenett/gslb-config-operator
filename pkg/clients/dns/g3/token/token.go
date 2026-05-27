package token

import (
	"context"
	"fmt"

	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/transport"
)

type Client interface {
	SelfRegister(user, token string) (bool, error)
}

type g3tokenClient struct {
	transport transport.Transport
	path      string
}

func NewG3TokenClient(transport transport.Transport) *g3tokenClient {
	return &g3tokenClient{
		transport: transport,
		path:      "/token",
	}
}

func (tc *g3tokenClient) SelfRegister(user, token string) (bool, error) {
	err := tc.transport.PostJSON(
		context.Background(),
		tc.path,
		map[string]string{
			"user":  user,
			"token": token,
		},
		nil,
	)
	if err != nil {
		return false, fmt.Errorf("failed to self-register to DNS G3: %w", err)
	}

	return true, nil
}

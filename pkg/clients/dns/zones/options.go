package zones

type ReadOptions struct {
	FQDN           *string
	Infrastructure *string
}

type ReadOption func(*ReadOptions)

func WithFQDN(fqdn string) ReadOption {
	return func(ro *ReadOptions) {
		ro.FQDN = new(fqdn)
	}
}

func WithInfrastructure(infrastructure string) ReadOption {
	return func(ro *ReadOptions) {
		ro.Infrastructure = new(infrastructure)
	}
}

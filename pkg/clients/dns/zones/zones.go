package zones

type Zone struct {
	ID             string
	Name           string
	DefaultTTL     string
	Infrastructure string
}

type Client interface {
	Read(...ReadOption) (*Zone, error)
}

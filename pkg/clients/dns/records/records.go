package records

import (
	"codeberg.org/miekg/dns"
)

type RecordRequest struct {
	dns.RR
	RecordID       string
	Infrastructure string

	// TODO: is this necessary?
	//ZoneID         *int
}

type Client interface {
	Read(...ReadOption) ([]RecordRequest, error)
	Create(RecordRequest) error
	Update(id string, req RecordRequest) error
	Delete(id string) error
}

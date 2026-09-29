package records

import (
	"fmt"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	"codeberg.org/miekg/dns"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/g3/models"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/records"
	"github.com/NorskHelsenett/gslb-config-operator/pkg/clients/dns/transport"
)

type G3RecordsAdapter struct {
	// native g3 client
	client Client
}

func NewG3RecordsAdapter(transport transport.Transport) *G3RecordsAdapter {
	return &G3RecordsAdapter{
		client: NewRecordsClient(transport),
	}
}

func (a *G3RecordsAdapter) Read(opts ...records.ReadOption) ([]records.RecordRequest, error) {
	readOptions := records.ReadOptions{}
	for _, opt := range opts {
		opt(&readOptions)
	}

	params := url.Values{}
	if readOptions.ZoneID != nil {
		params.Set("dnsZoneId", *readOptions.ZoneID)
	}

	if readOptions.RecordName != nil {
		params.Set("name", *readOptions.RecordName)
	}

	g3Page, err := a.client.Read(params)
	if err != nil {
		return nil, err
	}

	if g3Page == nil {
		return []records.RecordRequest{}, nil
	}

	records := make([]records.RecordRequest, 0, len(g3Page.Items))
	for _, item := range g3Page.Items {
		records = append(records, a.decode(item))
	}

	return records, nil
}

func (a *G3RecordsAdapter) Create(record records.RecordRequest) error {
	g3Record, err := a.encode(record)
	if err != nil {
		return fmt.Errorf("could create record: %w", err)
	}
	_, err = a.client.Create(g3Record)
	return err
}

func (a *G3RecordsAdapter) Update(id string, record records.RecordRequest) error {
	g3Record, err := a.encode(record)
	if err != nil {
		return fmt.Errorf("could not update record with id: %s: %w", id, err)
	}
	intID, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("failed to convert id to integer: %s: %w", id, err)
	}

	g3Record.Id = &intID
	_, err = a.client.Update(g3Record)
	return err
}

func (a *G3RecordsAdapter) Delete(id string) error {
	return a.client.Delete(id)
}

func (a *G3RecordsAdapter) encode(req records.RecordRequest) (models.G3Record, error) {
	hdr := req.RR.Header()
	record := models.G3Record{
		Name:           hdr.Name,
		Ttl:            strconv.FormatUint(uint64(hdr.TTL), 10),
		Infrastructure: req.Infrastructure,
	}
	switch v := req.RR.(type) {
	case *dns.A:
		record.Type = models.DNSRecordTypeA
		record.Address = v.A.String()
	case *dns.AAAA:
		record.Type = models.DNSRecordTypeAAAA
		record.Address = v.AAAA.String()
	case *dns.TXT:
		record.Type = models.DNSRecordTypeTXT
		record.Address = `"` + strings.ReplaceAll(strings.Join(v.Txt, ""), `"`, `\"`) + `"`

	case *dns.NS:
		record.Type = models.DNSRecordTypeNS
		record.Address = v.Ns
	default:
		return models.G3Record{}, fmt.Errorf("unsupported record type for G3: %T", req.RR)
	}
	return record, nil
}

func (a *G3RecordsAdapter) decode(record models.G3Record) records.RecordRequest {
	ttl, _ := strconv.ParseUint(record.Ttl, 10, 32)
	hdr := dns.Header{
		Name:  record.Name,
		Class: dns.ClassINET,
		TTL:   uint32(ttl),
	}

	var rr dns.RR
	switch record.Type {
	case models.DNSRecordTypeA:
		addr, _ := netip.ParseAddr(record.Address)
		rr = &dns.A{Hdr: hdr, Addr: addr.Unmap()}
	case models.DNSRecordTypeAAAA:
		addr, _ := netip.ParseAddr(record.Address)
		rr = &dns.AAAA{Hdr: hdr, Addr: addr}
	case models.DNSRecordTypeTXT:
		txt := strings.TrimPrefix(record.Address, `"`)
		txt = strings.TrimSuffix(txt, `"`)
		txt = strings.ReplaceAll(txt, `\"`, `"`)
		rr = &dns.TXT{Hdr: hdr, Txt: []string{txt}}
	case models.DNSRecordTypeNS:
		rr = &dns.NS{Hdr: hdr, Ns: record.Address}
	}

	var recordID string
	if record.Id != nil {
		id := strconv.Itoa(*record.Id)
		recordID = id
	}

	return records.RecordRequest{
		RR:             rr,
		RecordID:       recordID,
		Infrastructure: record.Infrastructure,
	}
}

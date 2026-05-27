package models

type G3RecordType string

const (
	DNSRecordTypeA    = "DnsRecordA"
	DNSRecordTypeAAAA = "DnsRecordAAAA"
	DNSRecordTypeTXT  = "DnsRecordTXT"
	DNSRecordTypeNS   = "DnsRecordNS"
)

const (
	DNSInfrastructureInternal   = "internal"
	DNSInfrastructureExternal   = "external"
	DNSInfrastructureDatacenter = "datacenter"
)

package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/netip"
)

type GSLBConfig struct {
	ServiceID        string     `json:"id"`
	MemberOf         string     `json:"memberOf"`
	Address          Address    `json:"address"`
	Port             string     `json:"port"`
	Path             *string    `json:"path,omitempty"`
	Datacenter       string     `json:"dc"`
	Views            []string   `json:"views,omitempty"`
	Interval         *string    `json:"interval,omitempty"`
	Priority         int        `json:"priority"`
	FailureThreshold *int       `json:"threshold,omitempty"`
	CheckType        string     `json:"check"`
	Script           *LuaScript `json:"lua,omitempty"`
}

type LuaScript string

func (s *LuaScript) UnmarshalJSON(b []byte) error {
	var encoded string
	if err := json.Unmarshal(b, &encoded); err != nil {
		*s = LuaScript("")
		return err
	}

	if encoded == "" {
		*s = LuaScript("")
		return nil
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		*s = LuaScript("")
		return fmt.Errorf("failed to decode base64 lua script: %w", err)
	}

	*s = LuaScript(decoded)

	return nil
}

func (s *LuaScript) MarshalJSON() ([]byte, error) {
	if s == nil || *s == "" {
		return json.Marshal("")
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(*s))

	return json.Marshal(encoded)
}

type Address struct {
	IPFamily string      `json:"ipFamily"`
	IP       *netip.Addr `json:"ip,omitempty"`
	IPv4     *netip.Addr `json:"ipv4,omitempty"`
	IPv6     *netip.Addr `json:"ipv6,omitempty"`
}

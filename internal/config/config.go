package config

import (
	"log"
	"log/slog"
	"strings"
)

var cfg *Config

type Config struct {
	Auth auth   `mapstructure:"auth"`
	SRV  server `mapstructure:"server"`
	Dns  dns    `mapstructure:"dns"`
}

func Server() *server {
	return &cfg.SRV
}

func Auth() *auth {
	return &cfg.Auth
}

func DNS() *dns {
	return &cfg.Dns
}

type auth struct {
	ISS string `mapstructure:"issuer"`
	AUD string `mapstructure:"audience"`
}

func (a *auth) Issuer() string {
	return a.ISS
}

func (a *auth) Audience() string {
	return a.AUD
}

type dns struct {
	ZONE         string   `mapstructure:"zone"`
	ZONE_INFRA   string   `mapstructure:"zoneInfra"`
	Generation   string   `mapstructure:"generation"`
	Updater      string   `mapstructure:"updater"`
	UpdaterCreds string   `mapstructure:"updaterCreds"`
	NAMESERVERS  []string `mapstructure:"nameServers"`
}

func (d *dns) Zone() string {
	return d.ZONE
}

func (d *dns) ZoneInfra() string {
	return d.ZONE_INFRA
}

type DNSGeneration string

const (
	G3 DNSGeneration = "G3"
	G4 DNSGeneration = "G4"
)

func (d *dns) Gen() DNSGeneration {
	if strings.Contains(d.Generation, "3") {
		return G3
	}
	return G4
}

func (d *dns) UpdaterURL() string {
	return "https://" + d.Updater
}

func (d *dns) Credentials() string {
	return d.UpdaterCreds
}

func (d *dns) NameServers() []string {
	return d.NAMESERVERS
}

type server struct {
	DC        string `mapstructure:"datacenter"`
	ClusterID string `mapstructure:"clusterID"`
	LOG_LEVEL string `mapstructure:"logLevel"`
}

func (s *server) Datacenter() string {
	return s.DC
}

func (s *server) Cluster() string {
	return s.ClusterID
}

func (s *server) LogLevel() slog.Level {
	switch s.LOG_LEVEL {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "info", "INFO":
		return slog.LevelInfo
	case "warn", "WARN":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelDebug
	}
}

func init() {
	err := Load()
	if err != nil {
		log.Panic("failed to load configuration", err)
	}
}

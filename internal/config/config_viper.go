package config

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/spf13/viper"
)

var secretsKeyMap = map[string]string{
    "DNS_UPDATER_CREDS": "dns.updaterCreds",
}

// Load initialises the package-level config. Call once at startup before using
// Server(), Auth(), or DNS().
func Load() error {
    c, err := new()
    if err != nil {
        return err
    }
    cfg = c
    return nil
}

func new() (*Config, error) {
    v := viper.New()
    setDefaults(v)

    cfgPath := os.Getenv("GSLBCONFIG")
    v.SetConfigName("config")
    v.SetConfigType("yaml")
    v.AddConfigPath(cfgPath)
    v.AddConfigPath(".")
    v.AddConfigPath("/app")
    if err := v.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return nil, fmt.Errorf("config file: %w", err)
        }
    }

    if _, err := loadSecrets(v, "./secrets"); err != nil {
        return nil, err
    }

    var c Config
    if err := v.Unmarshal(&c); err != nil {
        return nil, fmt.Errorf("unmarshalling config: %w", err)
    }

    return &c, nil
}

func setDefaults(v *viper.Viper) {
    v.SetDefault("auth.issuer", "")
    v.SetDefault("auth.audience", "")

    v.SetDefault("server.datacenter", "")
    v.SetDefault("server.clusterID", "")
    v.SetDefault("server.logLevel", "INFO")

    v.SetDefault("dns.zone", "")
    v.SetDefault("dns.generation", "G4")
    v.SetDefault("dns.updater", "")
    v.SetDefault("dns.updaterCreds", "")
    v.SetDefault("dns.nameServers", []string{})
}

func loadSecrets(v *viper.Viper, dir string) (loaded int, err error) {
    entries, err := os.ReadDir(dir)
    if os.IsNotExist(err) {
        return loaded, nil
    }
    if err != nil {
        return loaded, fmt.Errorf("failed to load secrets directory: %w", err)
    }

    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }
        key, ok := secretsKeyMap[entry.Name()]
        if !ok {
            continue
        }
        raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
        if err != nil {
            return loaded, fmt.Errorf("reading secret %s: %w", entry.Name(), err)
        }
        value := strings.TrimSpace(string(raw))
        if value != "" {
            v.Set(key, value)
            loaded++
        }
    }

    return loaded, nil
}
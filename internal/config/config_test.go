package config

import (
	"os"
	"testing"
)

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("INFLUXDB_URL", "http://localhost:8086")
	os.Setenv("INFLUXDB_TOKEN", "test-token")
	os.Setenv("INFLUXDB_ORG", "test-org")
	os.Setenv("INFLUXDB_BUCKET", "tokens")
	os.Setenv("PRICING_CONFIG", "../../deploy/config/pricing.yaml")
	os.Setenv("PORT", "9090")
	defer func() {
		os.Unsetenv("INFLUXDB_URL")
		os.Unsetenv("INFLUXDB_TOKEN")
		os.Unsetenv("INFLUXDB_ORG")
		os.Unsetenv("INFLUXDB_BUCKET")
		os.Unsetenv("PRICING_CONFIG")
		os.Unsetenv("PORT")
	}()

	cfg := Load()

	if cfg.InfluxURL != "http://localhost:8086" {
		t.Errorf("InfluxURL = %q, want %q", cfg.InfluxURL, "http://localhost:8086")
	}
	if cfg.InfluxToken != "test-token" {
		t.Errorf("InfluxToken = %q, want %q", cfg.InfluxToken, "test-token")
	}
	if cfg.InfluxOrg != "test-org" {
		t.Errorf("InfluxOrg = %q, want %q", cfg.InfluxOrg, "test-org")
	}
	if cfg.InfluxBucket != "tokens" {
		t.Errorf("InfluxBucket = %q, want %q", cfg.InfluxBucket, "tokens")
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
}

func TestLoadDefaults(t *testing.T) {
	os.Unsetenv("PORT")
	cfg := Load()
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want default %q", cfg.Port, "8080")
	}
}

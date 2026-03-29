package config

import "os"

type Config struct {
	InfluxURL    string
	InfluxToken  string
	InfluxOrg    string
	InfluxBucket string
	PricingPath  string
	RoutesPath   string
	Port         string
	RecorderPath string
}

func Load() Config {
	return Config{
		InfluxURL:    envOr("INFLUXDB_URL", "http://localhost:8086"),
		InfluxToken:  envOr("INFLUXDB_TOKEN", ""),
		InfluxOrg:    envOr("INFLUXDB_ORG", ""),
		InfluxBucket: envOr("INFLUXDB_BUCKET", "tokens"),
		PricingPath:  envOr("PRICING_CONFIG", "pricing.yaml"),
		RoutesPath:   envOr("ROUTES_CONFIG", "routes.yaml"),
		Port:         envOr("PORT", "8080"),
		RecorderPath: envOr("RECORDER_PATH", ""),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

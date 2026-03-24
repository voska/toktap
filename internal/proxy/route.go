package proxy

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Route struct {
	Provider            string   `yaml:"provider"`
	Upstream            *url.URL `yaml:"-"`
	InjectStreamOptions bool     `yaml:"inject_stream_options"`
	ChromeTransport     bool     `yaml:"chrome_transport"`
}

type routeYAML struct {
	Upstream            string `yaml:"upstream"`
	Provider            string `yaml:"provider"`
	InjectStreamOptions bool   `yaml:"inject_stream_options"`
	ChromeTransport     bool   `yaml:"chrome_transport"`
}

func LoadRoutes(path string) (map[string]*Route, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading routes config: %w", err)
	}
	var cfg struct {
		Routes map[string]routeYAML `yaml:"routes"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing routes config: %w", err)
	}
	routes := make(map[string]*Route, len(cfg.Routes))
	for name, raw := range cfg.Routes {
		u, err := url.Parse(raw.Upstream)
		if err != nil {
			return nil, fmt.Errorf("invalid upstream URL for route %q: %w", name, err)
		}
		if raw.Provider == "" {
			return nil, fmt.Errorf("route %q: provider is required", name)
		}
		routes[name] = &Route{
			Provider:            raw.Provider,
			Upstream:            u,
			InjectStreamOptions: raw.InjectStreamOptions,
			ChromeTransport:     raw.ChromeTransport,
		}
	}
	return routes, nil
}

func splitRoutePath(path string) (route, rest string) {
	path = strings.TrimPrefix(path, "/")
	if i := strings.IndexByte(path, '/'); i >= 0 {
		return path[:i], path[i:]
	}
	return path, "/"
}

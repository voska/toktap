package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/voska/toktap/internal/config"
	"github.com/voska/toktap/internal/pricing"
	"github.com/voska/toktap/internal/proxy"
)

func TestDefaultConfigFromEmptyDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("ROUTES_CONFIG", "")
	t.Setenv("PRICING_CONFIG", "")
	cfg := config.Load()

	routes, err := proxy.LoadRoutes(cfg.RoutesPath)
	if err != nil {
		t.Fatalf("load default routes: %v", err)
	}
	for _, name := range []string{"anthropic", "openai", "openrouter", "xai", "chatgpt", "openclaw"} {
		if routes[name] == nil {
			t.Errorf("missing default route %q", name)
		}
	}
	table, err := pricing.LoadFromFile(cfg.PricingPath)
	if err != nil {
		t.Fatalf("load default pricing: %v", err)
	}
	if err := table.Reload(cfg.PricingPath); err != nil {
		t.Fatalf("reload default pricing: %v", err)
	}
	if got := table.Calculate("gpt-4o", 1000000, 0, 0, 0); got != 2.5 {
		t.Errorf("default gpt-4o input cost = %v, want 2.5", got)
	}
}

func TestConfigFileOverrides(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		name := "working-directory"
		if explicit {
			name = "environment"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			t.Setenv("ROUTES_CONFIG", "")
			t.Setenv("PRICING_CONFIG", "")
			if explicit {
				t.Setenv("ROUTES_CONFIG", filepath.Join(dir, "routes.yaml"))
				t.Setenv("PRICING_CONFIG", filepath.Join(dir, "pricing.yaml"))
				t.Chdir(t.TempDir())
			}
			write := func(name, data string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			write("routes.yaml", "routes:\n  custom:\n    upstream: https://example.com\n    provider: openai\n")
			write("pricing.yaml", "models:\n  custom: {input_per_m: 1}\n")
			cfg := config.Load()
			routes, err := proxy.LoadRoutes(cfg.RoutesPath)
			if err != nil || len(routes) != 1 || routes["custom"] == nil {
				t.Fatalf("custom routes = %v, err = %v", routes, err)
			}
			table, err := pricing.LoadFromFile(cfg.PricingPath)
			if err != nil {
				t.Fatal(err)
			}
			if got := table.Calculate("custom", 1000000, 0, 0, 0); got != 1 {
				t.Fatalf("custom cost = %v, want 1", got)
			}
			write("pricing.yaml", "models:\n  custom: {input_per_m: 2}\n")
			if err := table.Reload(cfg.PricingPath); err != nil {
				t.Fatal(err)
			}
			if got := table.Calculate("custom", 1000000, 0, 0, 0); got != 2 {
				t.Fatalf("reloaded cost = %v, want 2", got)
			}
			if err := os.Remove(filepath.Join(dir, "pricing.yaml")); err != nil {
				t.Fatal(err)
			}
			if err := table.Reload(cfg.PricingPath); err == nil {
				t.Error("expected a reload error after removing custom pricing")
			}
			if got := table.Calculate("custom", 1000000, 0, 0, 0); got != 2 {
				t.Errorf("last good custom pricing changed after failed reload: %v", got)
			}
		})
	}
}

func TestConfigErrorsDoNotFallBack(t *testing.T) {
	for _, name := range []string{"missing-explicit", "malformed-working-directory", "directory-instead-of-file", "broken-symlink-working-directory"} {
		t.Run(name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv("ROUTES_CONFIG", "")
			t.Setenv("PRICING_CONFIG", "")
			switch name {
			case "missing-explicit":
				t.Setenv("ROUTES_CONFIG", "routes.yaml")
				t.Setenv("PRICING_CONFIG", "pricing.yaml")
			case "broken-symlink-working-directory":
				for _, path := range []string{"routes.yaml", "pricing.yaml"} {
					if err := os.Symlink("missing.yaml", path); err != nil {
						t.Fatal(err)
					}
				}
			case "directory-instead-of-file":
				for _, path := range []string{"routes.yaml", "pricing.yaml"} {
					if err := os.Mkdir(path, 0700); err != nil {
						t.Fatal(err)
					}
				}
			default:
				for _, path := range []string{"routes.yaml", "pricing.yaml"} {
					if err := os.WriteFile(path, []byte("[invalid yaml"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			cfg := config.Load()
			if _, err := proxy.LoadRoutes(cfg.RoutesPath); err == nil {
				t.Error("expected routes error, not embedded defaults")
			}
			if _, err := pricing.LoadFromFile(cfg.PricingPath); err == nil {
				t.Error("expected pricing error, not embedded defaults")
			}
		})
	}
}

package renderer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/clash-proxyd/proxyd/internal/parser"
	"github.com/clash-proxyd/proxyd/pkg/config"
)

// Renderer renders runtime mihomo configuration
type Renderer struct {
	policyCfg *config.PolicyConfig
	parser    *parser.Parser
}

// NewRenderer creates a new configuration renderer
func NewRenderer(policyCfg *config.PolicyConfig) *Renderer {
	return &Renderer{
		policyCfg: policyCfg,
		parser:    parser.NewParser(),
	}
}

// Render renders a complete mihomo configuration
func (r *Renderer) Render(sources []map[string]interface{}) (map[string]interface{}, error) {
	if len(sources) == 0 {
		return nil, fmt.Errorf("no sources to render")
	}

	// Merge all sources
	merged, err := r.parser.Merge(sources)
	if err != nil {
		return nil, fmt.Errorf("failed to merge sources: %w", err)
	}

	// Apply policy settings
	r.applyPolicy(merged)

	return merged, nil
}

// RenderToFile renders configuration and saves to file
func (r *Renderer) RenderToFile(sources []map[string]interface{}, path string) error {
	config, err := r.Render(sources)
	if err != nil {
		return err
	}

	// Convert to YAML
	yamlData, err := r.parser.ToYAML(config)
	if err != nil {
		return fmt.Errorf("failed to convert to YAML: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// applyPolicy applies policy settings to configuration
func (r *Renderer) applyPolicy(config map[string]interface{}) {
	// Apply basic settings
	if r.policyCfg.MixedPort > 0 {
		config["mixed-port"] = r.policyCfg.MixedPort
		delete(config, "port")       // Remove conflicting port config
		delete(config, "socks-port") // Remove conflicting socks-port config
	}
	if r.policyCfg.AllowLan {
		config["allow-lan"] = true
	}
	if r.policyCfg.BindAddress != "" {
		config["bind-address"] = r.policyCfg.BindAddress
	}
	if r.policyCfg.LogLevel != "" {
		config["log-level"] = r.policyCfg.LogLevel
	}
	if r.policyCfg.Mode != "" {
		config["mode"] = r.policyCfg.Mode
	}
	if r.policyCfg.ExternalController != "" {
		config["external-controller"] = r.policyCfg.ExternalController
	}
	config["ipv6"] = r.policyCfg.IPv6

	// Disable geodata (MMDB) to avoid download dependency on first run.
	// GEOIP rules will fall back gracefully without this database.
	config["geodata-mode"] = false

	// Ensure essential fields exist
	if _, ok := config["proxies"]; !ok {
		config["proxies"] = []interface{}{}
	}
	if _, ok := config["proxy-groups"]; !ok {
		config["proxy-groups"] = r.createDefaultGroups()
	}
	if _, ok := config["rules"]; !ok {
		config["rules"] = r.createDefaultRules()
	}
}

// createDefaultGroups creates default proxy groups
func (r *Renderer) createDefaultGroups() []interface{} {
	return []interface{}{
		map[string]interface{}{
			"name":    "Proxy",
			"type":    "select",
			"proxies": []string{"DIRECT"},
		},
		map[string]interface{}{
			"name":    "Auto",
			"type":    "url-test",
			"url":     "http://www.gstatic.com/generate_204",
			"interval": 300,
			"proxies": []string{"DIRECT"},
		},
	}
}

// createDefaultRules creates default rules
func (r *Renderer) createDefaultRules() []interface{} {
	return []interface{}{
		"DOMAIN-SUFFIX,local,DIRECT",
		"IP-CIDR,127.0.0.0/8,DIRECT",
		"IP-CIDR,172.16.0.0/12,DIRECT",
		"IP-CIDR,192.168.0.0/16,DIRECT",
		"MATCH,Proxy",
	}
}

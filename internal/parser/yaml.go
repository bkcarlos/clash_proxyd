package parser

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parser parses mihomo YAML configuration
type Parser struct{}

// NewParser creates a new parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses mihomo configuration from YAML data
func (p *Parser) Parse(data []byte) (map[string]interface{}, error) {
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate required fields
	if err := p.validate(config); err != nil {
		return nil, err
	}

	return config, nil
}

// ParseFile parses a mihomo configuration file
func (p *Parser) ParseFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return p.Parse(data)
}

// validate validates the mihomo configuration
func (p *Parser) validate(config map[string]interface{}) error {
	// Check for proxies field
	if _, ok := config["proxies"]; !ok {
		// proxies field is optional for local configs
	}

	// Check for proxy-groups field
	if _, ok := config["proxy-groups"]; !ok {
		// proxy-groups is optional
	}

	// Check for rules field
	if _, ok := config["rules"]; !ok {
		// rules is optional
	}

	return nil
}

// GetProxies extracts proxies from configuration
func (p *Parser) GetProxies(config map[string]interface{}) ([]interface{}, error) {
	proxies, ok := config["proxies"]
	if !ok {
		return []interface{}{}, nil
	}

	proxyList, ok := proxies.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid proxies format")
	}

	return proxyList, nil
}

// GetProxyGroups extracts proxy groups from configuration
func (p *Parser) GetProxyGroups(config map[string]interface{}) ([]interface{}, error) {
	groups, ok := config["proxy-groups"]
	if !ok {
		return []interface{}{}, nil
	}

	groupList, ok := groups.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid proxy-groups format")
	}

	return groupList, nil
}

// GetRules extracts rules from configuration
func (p *Parser) GetRules(config map[string]interface{}) ([]interface{}, error) {
	rules, ok := config["rules"]
	if !ok {
		return []interface{}{}, nil
	}

	ruleList, ok := rules.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid rules format")
	}

	return ruleList, nil
}

// Merge merges multiple configurations
func (p *Parser) Merge(configs []map[string]interface{}) (map[string]interface{}, error) {
	if len(configs) == 0 {
		return nil, fmt.Errorf("no configs to merge")
	}

	if len(configs) == 1 {
		return configs[0], nil
	}

	result := make(map[string]interface{})
	allProxies := []interface{}{}
	allGroups := []interface{}{}
	allRules := []interface{}{}

	for _, config := range configs {
		// Merge basic fields from first config
		for k, v := range config {
			if k == "proxies" || k == "proxy-groups" || k == "rules" {
				continue
			}
			if _, exists := result[k]; !exists {
				result[k] = v
			}
		}

		// Collect proxies
		if proxies, err := p.GetProxies(config); err == nil {
			allProxies = append(allProxies, proxies...)
		}

		// Collect groups
		if groups, err := p.GetProxyGroups(config); err == nil {
			allGroups = append(allGroups, groups...)
		}

		// Collect rules
		if rules, err := p.GetRules(config); err == nil {
			allRules = append(allRules, rules...)
		}
	}

	if len(allProxies) > 0 {
		result["proxies"] = allProxies
	}
	if len(allGroups) > 0 {
		result["proxy-groups"] = allGroups
	}
	if len(allRules) > 0 {
		result["rules"] = allRules
	}

	return result, nil
}

// ToYAML converts configuration map to YAML bytes
func (p *Parser) ToYAML(config map[string]interface{}) ([]byte, error) {
	data, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return data, nil
}

// ToYAMLString converts configuration map to YAML string
func (p *Parser) ToYAMLString(config map[string]interface{}) (string, error) {
	data, err := p.ToYAML(config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// IsSubscriptionURL checks if a string is a subscription URL
func IsSubscriptionURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}

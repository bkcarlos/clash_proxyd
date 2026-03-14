package policy

import (
	"fmt"
	"sort"
	"strings"

	"github.com/clash-proxyd/proxyd/internal/parser"
)

// Generator generates mihomo policy configurations
type Generator struct {
	parser *parser.Parser
}

// NewGenerator creates a new policy generator
func NewGenerator() *Generator {
	return &Generator{
		parser: parser.NewParser(),
	}
}

// GroupConfig represents a proxy group configuration
type GroupConfig struct {
	Name    string
	Type    string
	Proxies []string
	URL     string
	Interval int
}

// RuleConfig represents a rule configuration
type RuleConfig struct {
	Type    string
	Value   string
	Policy  string
}

// GenerateGroups generates proxy groups
func (g *Generator) GenerateGroups(proxies []string) []interface{} {
	// Remove duplicates
	uniqueProxies := make(map[string]bool)
	for _, p := range proxies {
		uniqueProxies[p] = true
	}

	proxyList := make([]string, 0, len(uniqueProxies))
	for p := range uniqueProxies {
		proxyList = append(proxyList, p)
	}
	sort.Strings(proxyList)

	groups := []interface{}{}

	// Create main select group
	groups = append(groups, map[string]interface{}{
		"name":    "Proxy",
		"type":    "select",
		"proxies": append([]string{"Auto", "DIRECT"}, proxyList...),
	})

	// Create auto test group
	if len(proxyList) > 0 {
		groups = append(groups, map[string]interface{}{
			"name":     "Auto",
			"type":     "url-test",
			"url":      "http://www.gstatic.com/generate_204",
			"interval": 300,
			"proxies":  proxyList,
		})
	}

	// Create fallback group
	if len(proxyList) > 0 {
		groups = append(groups, map[string]interface{}{
			"name":     "Fallback",
			"type":     "fallback",
			"url":      "http://www.gstatic.com/generate_204",
			"interval": 300,
			"proxies":  proxyList,
		})
	}

	// Create load balance group
	if len(proxyList) > 0 {
		groups = append(groups, map[string]interface{}{
			"name":    "LoadBalance",
			"type":    "load-balance",
			"url":     "http://www.gstatic.com/generate_204",
			"interval": 300,
			"proxies": proxyList,
		})
	}

	return groups
}

// GenerateRules generates rule configurations
func (g *Generator) GenerateRules(customRules []string) []interface{} {
	rules := []interface{}{}

	// Add default rules
	rules = append(rules, "DOMAIN-SUFFIX,local,DIRECT")
	rules = append(rules, "IP-CIDR,127.0.0.0/8,DIRECT")
	rules = append(rules, "IP-CIDR,172.16.0.0/12,DIRECT")
	rules = append(rules, "IP-CIDR,192.168.0.0/16,DIRECT")
	rules = append(rules, "IP-CIDR,10.0.0.0/8,DIRECT")

	// Add custom rules
	for _, rule := range customRules {
		rule = strings.TrimSpace(rule)
		if rule != "" && !strings.HasPrefix(rule, "#") {
			rules = append(rules, rule)
		}
	}

	// Add final match rule
	rules = append(rules, "MATCH,Proxy")

	return rules
}

// GenerateCustomGroup generates a custom proxy group
func (g *Generator) GenerateCustomGroup(config GroupConfig) (interface{}, error) {
	if config.Name == "" {
		return nil, fmt.Errorf("group name is required")
	}
	if config.Type == "" {
		return nil, fmt.Errorf("group type is required")
	}

	group := map[string]interface{}{
		"name":    config.Name,
		"type":    config.Type,
		"proxies": config.Proxies,
	}

	if config.URL != "" {
		group["url"] = config.URL
	}
	if config.Interval > 0 {
		group["interval"] = config.Interval
	}

	return group, nil
}

// MergeGroups merges multiple proxy group configurations
func (g *Generator) MergeGroups(groupsList ...[]interface{}) []interface{} {
	allGroups := []interface{}{}
	for _, groups := range groupsList {
		allGroups = append(allGroups, groups...)
	}
	return allGroups
}

// MergeRules merges multiple rule configurations
func (g *Generator) MergeRules(rulesList ...[]interface{}) []interface{} {
	allRules := []interface{}{}
	for _, rules := range rulesList {
		allRules = append(allRules, rules...)
	}
	return allRules
}

// ExtractProxies extracts proxy names from configuration
func (g *Generator) ExtractProxies(config map[string]interface{}) ([]string, error) {
	proxies, err := g.parser.GetProxies(config)
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, p := range proxies {
		if proxyMap, ok := p.(map[string]interface{}); ok {
			if name, ok := proxyMap["name"].(string); ok {
				names = append(names, name)
			}
		}
	}

	return names, nil
}

// ValidateRule validates a rule string
func (g *Generator) ValidateRule(rule string) error {
	parts := strings.Split(rule, ",")
	if len(parts) < 3 {
		return fmt.Errorf("invalid rule format: %s", rule)
	}

	ruleType := parts[0]
	validTypes := map[string]bool{
		"DOMAIN": true, "DOMAIN-SUFFIX": true, "DOMAIN-KEYWORD": true,
		"IP-CIDR": true, "SRC-IP-CIDR": true, "GEOIP": true,
		"DST-PORT": true, "SRC-PORT": true, "MATCH": true,
		"RULE-SET": true,
	}

	if !validTypes[ruleType] {
		return fmt.Errorf("invalid rule type: %s", ruleType)
	}

	return nil
}

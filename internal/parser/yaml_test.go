package parser

import (
	"testing"
)

func TestParserParse(t *testing.T) {
	p := NewParser()

	t.Run("parse valid yaml", func(t *testing.T) {
		cfg, err := p.Parse([]byte("proxies:\n  - name: p1\n    type: ss\nproxy-groups:\n  - name: g1\n    type: select\n    proxies: [p1]\nrules:\n  - MATCH,g1\n"))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if _, ok := cfg["proxies"]; !ok {
			t.Fatalf("expected proxies in parsed config")
		}
	})

	t.Run("parse invalid yaml", func(t *testing.T) {
		_, err := p.Parse([]byte("proxies: ["))
		if err == nil {
			t.Fatalf("expected parse error for invalid yaml")
		}
	})
}

func TestParserMerge(t *testing.T) {
	p := NewParser()

	t.Run("merge multiple configs", func(t *testing.T) {
		cfg1 := map[string]interface{}{
			"port": 7890,
			"proxies": []interface{}{
				map[string]interface{}{"name": "p1"},
			},
			"proxy-groups": []interface{}{
				map[string]interface{}{"name": "g1"},
			},
		}
		cfg2 := map[string]interface{}{
			"proxies": []interface{}{
				map[string]interface{}{"name": "p2"},
			},
			"rules": []interface{}{"MATCH,g1"},
		}

		merged, err := p.Merge([]map[string]interface{}{cfg1, cfg2})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		proxies, err := p.GetProxies(merged)
		if err != nil {
			t.Fatalf("expected valid proxies, got %v", err)
		}
		if len(proxies) != 2 {
			t.Fatalf("expected 2 proxies, got %d", len(proxies))
		}

		rules, err := p.GetRules(merged)
		if err != nil {
			t.Fatalf("expected valid rules, got %v", err)
		}
		if len(rules) != 1 {
			t.Fatalf("expected 1 rule, got %d", len(rules))
		}
	})

	t.Run("merge empty config list", func(t *testing.T) {
		_, err := p.Merge(nil)
		if err == nil {
			t.Fatalf("expected error when merging empty configs")
		}
	})
}

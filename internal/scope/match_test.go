package scope

import (
	"testing"

	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/targets"
)

func TestEvaluateTargetAllowsWhenNoIncludeRules(t *testing.T) {
	decision := evaluate(t, "example.com", nil)
	if !decision.Allowed {
		t.Fatalf("expected target allowed without include rules: %+v", decision)
	}
}

func TestEvaluateTargetRequiresIncludeMatch(t *testing.T) {
	rules := []sqlite.ScopeRule{includeRule("rule_1", sqlite.ScopeTargetDomain, "example.com")}

	if decision := evaluate(t, "example.com", rules); !decision.Allowed {
		t.Fatalf("expected matching target allowed: %+v", decision)
	}
	if decision := evaluate(t, "other.com", rules); decision.Allowed {
		t.Fatalf("expected non-matching target blocked: %+v", decision)
	}
}

func TestEvaluateTargetExcludeOverridesInclude(t *testing.T) {
	rules := []sqlite.ScopeRule{
		includeRule("include_1", sqlite.ScopeTargetDomain, "*.example.com"),
		excludeRule("exclude_1", sqlite.ScopeTargetDomain, "admin.example.com"),
	}

	if decision := evaluate(t, "api.example.com", rules); !decision.Allowed {
		t.Fatalf("expected included subdomain allowed: %+v", decision)
	}
	if decision := evaluate(t, "admin.example.com", rules); decision.Allowed {
		t.Fatalf("expected excluded subdomain blocked: %+v", decision)
	}
}

func TestEvaluateTargetMatchesWildcardDomains(t *testing.T) {
	rules := []sqlite.ScopeRule{includeRule("rule_1", sqlite.ScopeTargetWildcard, "*.example.com")}

	if decision := evaluate(t, "api.example.com", rules); !decision.Allowed {
		t.Fatalf("expected wildcard subdomain allowed: %+v", decision)
	}
	if decision := evaluate(t, "example.com", rules); decision.Allowed {
		t.Fatalf("expected wildcard not to match apex: %+v", decision)
	}
}

func TestEvaluateTargetMatchesURLHostAgainstDomainRule(t *testing.T) {
	rules := []sqlite.ScopeRule{includeRule("rule_1", sqlite.ScopeTargetDomain, "*.example.com")}

	if decision := evaluate(t, "https://api.example.com/path", rules); !decision.Allowed {
		t.Fatalf("expected URL host to match domain rule: %+v", decision)
	}
}

func TestEvaluateTargetMatchesIPAndCIDRRules(t *testing.T) {
	if decision := evaluate(t, "1.2.3.4", []sqlite.ScopeRule{includeRule("rule_1", sqlite.ScopeTargetIP, "1.2.3.4")}); !decision.Allowed {
		t.Fatalf("expected exact IP allowed: %+v", decision)
	}
	if decision := evaluate(t, "10.0.0.42", []sqlite.ScopeRule{includeRule("rule_2", sqlite.ScopeTargetCIDR, "10.0.0.0/24")}); !decision.Allowed {
		t.Fatalf("expected IP inside CIDR allowed: %+v", decision)
	}
	if decision := evaluate(t, "10.0.1.1", []sqlite.ScopeRule{includeRule("rule_3", sqlite.ScopeTargetCIDR, "10.0.0.0/24")}); decision.Allowed {
		t.Fatalf("expected IP outside CIDR blocked: %+v", decision)
	}
}

func TestEvaluateTargetMatchesExactURLAndServiceRules(t *testing.T) {
	if decision := evaluate(t, "https://example.com/path", []sqlite.ScopeRule{includeRule("rule_1", sqlite.ScopeTargetURL, "https://example.com/path/")}); !decision.Allowed {
		t.Fatalf("expected exact URL allowed: %+v", decision)
	}
	if decision := evaluate(t, "example.com:443", []sqlite.ScopeRule{includeRule("rule_2", sqlite.ScopeTargetService, "example.com:443")}); !decision.Allowed {
		t.Fatalf("expected exact service allowed: %+v", decision)
	}
}

func evaluate(t *testing.T, raw string, rules []sqlite.ScopeRule) Decision {
	t.Helper()
	target, err := targets.Parse(raw)
	if err != nil {
		t.Fatalf("parse target %q: %v", raw, err)
	}
	return EvaluateTarget(target, rules)
}

func includeRule(id string, targetType sqlite.ScopeTargetType, value string) sqlite.ScopeRule {
	return sqlite.ScopeRule{ID: id, Effect: sqlite.ScopeEffectInclude, TargetType: targetType, Value: value}
}

func excludeRule(id string, targetType sqlite.ScopeTargetType, value string) sqlite.ScopeRule {
	return sqlite.ScopeRule{ID: id, Effect: sqlite.ScopeEffectExclude, TargetType: targetType, Value: value}
}

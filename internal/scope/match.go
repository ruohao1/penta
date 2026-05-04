package scope

import (
	"fmt"
	"net"
	"strings"

	"github.com/ruohao1/penta/internal/storage/sqlite"
	"github.com/ruohao1/penta/internal/targets"
)

type Decision struct {
	Allowed bool
	Reason  string
}

func EvaluateTarget(target targets.Target, rules []sqlite.ScopeRule) Decision {
	if target == nil {
		return Decision{Allowed: false, Reason: "target is required"}
	}
	includeCount := 0
	for _, rule := range rules {
		if rule.Effect == sqlite.ScopeEffectInclude {
			includeCount++
		}
		if rule.Effect == sqlite.ScopeEffectExclude && ruleMatchesTarget(rule, target) {
			return Decision{Allowed: false, Reason: fmt.Sprintf("target %s is excluded by scope rule %s", target.String(), rule.ID)}
		}
	}
	if includeCount == 0 {
		return Decision{Allowed: true, Reason: "no include scope rules configured"}
	}
	for _, rule := range rules {
		if rule.Effect == sqlite.ScopeEffectInclude && ruleMatchesTarget(rule, target) {
			return Decision{Allowed: true, Reason: fmt.Sprintf("target %s is included by scope rule %s", target.String(), rule.ID)}
		}
	}
	return Decision{Allowed: false, Reason: fmt.Sprintf("target %s is not included in session scope", target.String())}
}

func ruleMatchesTarget(rule sqlite.ScopeRule, target targets.Target) bool {
	switch rule.TargetType {
	case sqlite.ScopeTargetDomain:
		return matchesDomainRule(rule.Value, target)
	case sqlite.ScopeTargetWildcard:
		return matchesWildcardDomain(rule.Value, target)
	case sqlite.ScopeTargetIP:
		return matchesIPRule(rule.Value, target)
	case sqlite.ScopeTargetCIDR:
		return matchesCIDRRule(rule.Value, target)
	case sqlite.ScopeTargetURL:
		return target.Type() == targets.TypeURL && normalizeURL(rule.Value) == normalizeURL(target.String())
	case sqlite.ScopeTargetService:
		return target.Type() == targets.TypeService && strings.EqualFold(rule.Value, target.String())
	default:
		return false
	}
}

func matchesDomainRule(value string, target targets.Target) bool {
	if strings.HasPrefix(value, "*.") {
		return matchesWildcardDomain(value, target)
	}
	host, ok := targetHost(target)
	if !ok {
		return false
	}
	return normalizeDomain(host) == normalizeDomain(value)
}

func matchesWildcardDomain(value string, target targets.Target) bool {
	host, ok := targetHost(target)
	if !ok {
		return false
	}
	base := strings.TrimPrefix(normalizeDomain(value), "*.")
	host = normalizeDomain(host)
	return host != base && strings.HasSuffix(host, "."+base)
}

func matchesIPRule(value string, target targets.Target) bool {
	ruleIP := net.ParseIP(value)
	targetIP := targetIP(target)
	return ruleIP != nil && targetIP != nil && ruleIP.Equal(targetIP)
}

func matchesCIDRRule(value string, target targets.Target) bool {
	_, network, err := net.ParseCIDR(value)
	if err != nil {
		return false
	}
	if target.Type() == targets.TypeIP || target.Type() == targets.TypeService {
		ip := targetIP(target)
		return ip != nil && network.Contains(ip)
	}
	if target.Type() == targets.TypeCIDR {
		ip, _, err := net.ParseCIDR(target.String())
		return err == nil && network.Contains(ip)
	}
	return false
}

func targetIP(target targets.Target) net.IP {
	switch typed := target.(type) {
	case *targets.IP:
		return net.ParseIP(typed.String())
	case *targets.Service:
		if strings.EqualFold(typed.Host, "localhost") {
			return net.ParseIP("127.0.0.1")
		}
		return net.ParseIP(typed.Host)
	case *targets.URL:
		if strings.EqualFold(typed.Host, "localhost") {
			return net.ParseIP("127.0.0.1")
		}
		return net.ParseIP(typed.Host)
	default:
		return nil
	}
}

func targetHost(target targets.Target) (string, bool) {
	switch typed := target.(type) {
	case *targets.Domain:
		return typed.String(), true
	case *targets.URL:
		return typed.Host, true
	case *targets.Service:
		if net.ParseIP(typed.Host) != nil {
			return "", false
		}
		return typed.Host, true
	default:
		return "", false
	}
}

func normalizeDomain(value string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ".")
}

func normalizeURL(value string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), "/")
}

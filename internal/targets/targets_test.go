package targets

import "testing"

func TestParseDomain(t *testing.T) {
	target, err := Parse("example.com")
	if err != nil {
		t.Fatalf("parse domain: %v", err)
	}
	if got := target.String(); got != "example.com" {
		t.Fatalf("unexpected target value: got %q want %q", got, "example.com")
	}
	if got := target.Type(); got != TypeDomain {
		t.Fatalf("unexpected target type: got %q want %q", got, TypeDomain)
	}
}

func TestParseIP(t *testing.T) {
	target, err := Parse("1.2.3.4")
	if err != nil {
		t.Fatalf("parse ip: %v", err)
	}
	if got := target.String(); got != "1.2.3.4" {
		t.Fatalf("unexpected target value: got %q want %q", got, "1.2.3.4")
	}
	if got := target.Type(); got != TypeIP {
		t.Fatalf("unexpected target type: got %q want %q", got, TypeIP)
	}
}

func TestParseURL(t *testing.T) {
	target, err := Parse("https://example.com/foo?a=b")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	if got := target.String(); got != "https://example.com/foo?a=b" {
		t.Fatalf("unexpected target value: got %q want %q", got, "https://example.com/foo?a=b")
	}
	if got := target.Type(); got != TypeURL {
		t.Fatalf("unexpected target type: got %q want %q", got, TypeURL)
	}
}

func TestParseCIDR(t *testing.T) {
	target, err := Parse("10.0.0.0/24")
	if err != nil {
		t.Fatalf("parse cidr: %v", err)
	}
	if got := target.String(); got != "10.0.0.0/24" {
		t.Fatalf("unexpected target value: got %q want %q", got, "10.0.0.0/24")
	}
	if got := target.Type(); got != TypeCIDR {
		t.Fatalf("unexpected target type: got %q want %q", got, TypeCIDR)
	}
}

func TestParseService(t *testing.T) {
	target, err := Parse("example.com:443")
	if err != nil {
		t.Fatalf("parse service: %v", err)
	}
	if got := target.String(); got != "example.com:443" {
		t.Fatalf("unexpected target value: got %q want %q", got, "example.com:443")
	}
	if got := target.Type(); got != TypeService {
		t.Fatalf("unexpected target type: got %q want %q", got, TypeService)
	}
}

func TestParseLocalhostService(t *testing.T) {
	target, err := Parse("localhost:8000")
	if err != nil {
		t.Fatalf("parse localhost service: %v", err)
	}
	if got := target.String(); got != "localhost:8000" {
		t.Fatalf("unexpected target value: got %q want %q", got, "localhost:8000")
	}
	if got := target.Type(); got != TypeService {
		t.Fatalf("unexpected target type: got %q want %q", got, TypeService)
	}
}

func TestParseIPRange(t *testing.T) {
	target, err := Parse("1-255.1-255.1-255.1-255")
	if err != nil {
		t.Fatalf("parse ip range: %v", err)
	}
	if got := target.String(); got != "1-255.1-255.1-255.1-255" {
		t.Fatalf("unexpected target value: got %q want %q", got, "1-255.1-255.1-255.1-255")
	}
	if got := target.Type(); got != TypeIPRange {
		t.Fatalf("unexpected target type: got %q want %q", got, TypeIPRange)
	}
}

func TestParseRejectsEmptyTarget(t *testing.T) {
	if _, err := Parse("   "); err == nil {
		t.Fatal("expected empty target to fail")
	}
}

func TestParseRejectsInvalidServicePort(t *testing.T) {
	if _, err := Parse("example.com:notaport"); err == nil {
		t.Fatal("expected invalid service to fail")
	}
}

func TestCIDRContainsIP(t *testing.T) {
	target, err := Parse("10.0.0.0/24")
	if err != nil {
		t.Fatalf("parse cidr: %v", err)
	}
	cidr, ok := target.(*CIDR)
	if !ok {
		t.Fatalf("unexpected target type %T", target)
	}

	ipTarget, err := Parse("10.0.0.42")
	if err != nil {
		t.Fatalf("parse ip: %v", err)
	}
	ip, ok := ipTarget.(*IP)
	if !ok {
		t.Fatalf("unexpected ip target type %T", ipTarget)
	}

	if !cidr.Contains(ip) {
		t.Fatal("expected cidr to contain ip")
	}

	otherTarget, err := Parse("10.0.1.1")
	if err != nil {
		t.Fatalf("parse ip outside cidr: %v", err)
	}
	otherIP, ok := otherTarget.(*IP)
	if !ok {
		t.Fatalf("unexpected ip target type %T", otherTarget)
	}
	if cidr.Contains(otherIP) {
		t.Fatal("expected cidr not to contain outside ip")
	}
}

func TestIPRangeContainsIP(t *testing.T) {
	target, err := Parse("1-10.2.3.4-8")
	if err != nil {
		t.Fatalf("parse ip range: %v", err)
	}
	ipRange, ok := target.(*IPRange)
	if !ok {
		t.Fatalf("unexpected target type %T", target)
	}

	ipTarget, err := Parse("5.2.3.6")
	if err != nil {
		t.Fatalf("parse matching ip: %v", err)
	}
	ip, ok := ipTarget.(*IP)
	if !ok {
		t.Fatalf("unexpected ip target type %T", ipTarget)
	}
	if !ipRange.Contains(ip) {
		t.Fatal("expected ip range to contain matching ip")
	}

	otherTarget, err := Parse("11.2.3.6")
	if err != nil {
		t.Fatalf("parse non-matching ip: %v", err)
	}
	otherIP, ok := otherTarget.(*IP)
	if !ok {
		t.Fatalf("unexpected ip target type %T", otherTarget)
	}
	if ipRange.Contains(otherIP) {
		t.Fatal("expected ip range not to contain non-matching ip")
	}
}

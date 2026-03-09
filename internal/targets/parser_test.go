package targets

import "testing"

func TestParseOneIP(t *testing.T) {
	target, err := ParseOne("192.168.1.10")
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if target.Kind != KindIP {
		t.Fatalf("kind = %q, want %q", target.Kind, KindIP)
	}
	if got := target.IP.String(); got != "192.168.1.10" {
		t.Fatalf("ip = %q, want %q", got, "192.168.1.10")
	}
}

func TestParseOneIPv6(t *testing.T) {
	target, err := ParseOne("2001:db8::1")
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if target.Kind != KindIP {
		t.Fatalf("kind = %q, want %q", target.Kind, KindIP)
	}
	if got := target.IP.String(); got != "2001:db8::1" {
		t.Fatalf("ip = %q, want %q", got, "2001:db8::1")
	}
}

func TestParseOneCIDRMasked(t *testing.T) {
	target, err := ParseOne("10.0.0.7/24")
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if target.Kind != KindCIDR {
		t.Fatalf("kind = %q, want %q", target.Kind, KindCIDR)
	}
	if got := target.CIDR.String(); got != "10.0.0.0/24" {
		t.Fatalf("cidr = %q, want %q", got, "10.0.0.0/24")
	}
}

func TestParseOneURL(t *testing.T) {
	target, err := ParseOne("https://example.com/admin")
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if target.Kind != KindURL {
		t.Fatalf("kind = %q, want %q", target.Kind, KindURL)
	}
	if got := target.URL.String(); got != "https://example.com/admin" {
		t.Fatalf("url = %q, want %q", got, "https://example.com/admin")
	}
}

func TestParseOneRejectsUnsupportedScheme(t *testing.T) {
	_, err := ParseOne("ftp://example.com")
	if err == nil {
		t.Fatal("expected error for unsupported scheme")
	}
}

func TestParseOneRejectsInvalidFormat(t *testing.T) {
	_, err := ParseOne("example.com")
	if err == nil {
		t.Fatal("expected error for unsupported target format")
	}
}

func TestParseMany(t *testing.T) {
	targets, err := ParseMany([]string{"192.168.1.1", "10.0.0.0/30", "http://example.com"})
	if err != nil {
		t.Fatalf("ParseMany() error = %v", err)
	}
	if len(targets) != 3 {
		t.Fatalf("len = %d, want 3", len(targets))
	}
}

func TestAssertKind(t *testing.T) {
	ip, err := ParseOne("192.168.1.10")
	if err != nil {
		t.Fatalf("ParseOne() error = %v", err)
	}
	if err := ip.AssertKind(KindIP); err != nil {
		t.Fatalf("AssertKind(ip) error = %v", err)
	}
	if err := ip.AssertKind(KindCIDR); err == nil {
		t.Fatal("expected kind mismatch error")
	}
}

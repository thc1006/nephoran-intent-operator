package security

import (
	"net"
	"testing"
)

func TestValidateEndpointURL_BlocksDangerousSchemes(t *testing.T) {
	v := NewDefaultSSRFValidator()

	cases := []struct {
		name string
		url  string
	}{
		{"file scheme", "file:///etc/passwd"},
		{"file scheme windows", "file:///C:/Windows/System32/config/SAM"},
		{"gopher scheme", "gopher://evil.com:70/_1"},
		{"ftp scheme", "ftp://evil.com/secret"},
		{"dict scheme", "dict://evil.com:1234/info"},
		{"ldap scheme", "ldap://evil.com/cn=admin"},
		{"ssh scheme", "ssh://evil.com"},
		{"telnet scheme", "telnet://evil.com"},
		{"data scheme", "data:text/html,<h1>test</h1>"},
		{"javascript scheme", "javascript:alert(1)"},
		{"no scheme", "evil.com/path"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateEndpointURL(tc.url)
			if err == nil {
				t.Errorf("expected error for %q, got nil", tc.url)
			}
			if _, ok := err.(*SSRFError); !ok {
				t.Errorf("expected *SSRFError, got %T", err)
			}
		})
	}
}

func TestValidateEndpointURL_BlocksLocalhost(t *testing.T) {
	v := NewDefaultSSRFValidator()

	cases := []struct {
		name string
		url  string
	}{
		{"localhost", "http://localhost/api"},
		{"localhost with port", "http://localhost:8080/api"},
		{"localhost FQDN", "http://localhost./api"},
		{"localhost.localdomain", "http://localhost.localdomain/api"},
		{"ip6-localhost", "http://ip6-localhost/api"},
		{"ip6-loopback", "http://ip6-loopback/api"},
		{"127.0.0.1", "http://127.0.0.1/api"},
		{"127.0.0.1 with port", "http://127.0.0.1:9090/api"},
		{"127.1.2.3", "http://127.1.2.3/api"},
		{"0.0.0.0", "http://0.0.0.0/api"},
		{"IPv6 loopback", "http://[::1]/api"},
		{"IPv6 loopback with port", "http://[::1]:8080/api"},
		{"unspecified IPv6", "http://[::]/api"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateEndpointURL(tc.url)
			if err == nil {
				t.Errorf("expected error for %q, got nil", tc.url)
			}
		})
	}
}

func TestValidateEndpointURL_BlocksPrivateIPs(t *testing.T) {
	v := NewDefaultSSRFValidator()

	cases := []struct {
		name string
		url  string
	}{
		{"10.0.0.1", "http://10.0.0.1/api"},
		{"10.255.255.255", "http://10.255.255.255/api"},
		{"172.16.0.1", "http://172.16.0.1/api"},
		{"172.31.255.255", "http://172.31.255.255/api"},
		{"192.168.0.1", "http://192.168.0.1/api"},
		{"192.168.1.100", "http://192.168.1.100:8080/api"},
		{"IPv6 ULA fc00", "http://[fc00::1]/api"},
		{"IPv6 ULA fd00", "http://[fd12:3456::1]/api"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateEndpointURL(tc.url)
			if err == nil {
				t.Errorf("expected error for %q, got nil", tc.url)
			}
		})
	}
}

func TestValidateEndpointURL_BlocksLinkLocal(t *testing.T) {
	v := NewDefaultSSRFValidator()

	cases := []struct {
		name string
		url  string
	}{
		{"169.254.0.1", "http://169.254.0.1/api"},
		{"169.254.169.254 metadata", "http://169.254.169.254/latest/meta-data/"},
		{"169.254.255.255", "http://169.254.255.255/api"},
		{"IPv6 link-local", "http://[fe80::1]/api"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateEndpointURL(tc.url)
			if err == nil {
				t.Errorf("expected error for %q, got nil", tc.url)
			}
		})
	}
}

func TestValidateEndpointURL_BlocksCloudMetadata(t *testing.T) {
	v := NewDefaultSSRFValidator()

	cases := []struct {
		name string
		url  string
	}{
		{"AWS metadata", "http://169.254.169.254/latest/meta-data/"},
		{"AWS metadata token", "http://169.254.169.254/latest/api/token"},
		{"GCP metadata", "http://metadata.google.internal/computeMetadata/v1/"},
		{"Azure metadata", "http://metadata.azure.com/metadata/instance"},
		{"AWS instance-data", "http://instance-data/latest/meta-data/"},
		{"K8s API server", "http://kubernetes.default.svc/api/v1/secrets"},
		{"K8s API short", "http://kubernetes.default/api/v1/secrets"},
		{"K8s API shortest", "http://kubernetes/api/v1/secrets"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateEndpointURL(tc.url)
			if err == nil {
				t.Errorf("expected error for %q, got nil", tc.url)
			}
		})
	}
}

func TestValidateEndpointURL_BlocksUserinfo(t *testing.T) {
	v := NewDefaultSSRFValidator()

	cases := []struct {
		name string
		url  string
	}{
		{"user in URL", "http://admin@evil.com/api"},
		{"user:pass in URL", "http://admin:secret@evil.com/api"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateEndpointURL(tc.url)
			if err == nil {
				t.Errorf("expected error for %q, got nil", tc.url)
			}
		})
	}
}

func TestValidateEndpointURL_BlocksEmptyAndMalformed(t *testing.T) {
	v := NewDefaultSSRFValidator()

	cases := []struct {
		name string
		url  string
	}{
		{"empty string", ""},
		{"only scheme", "http://"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateEndpointURL(tc.url)
			if err == nil {
				t.Errorf("expected error for %q, got nil", tc.url)
			}
		})
	}
}

func TestValidateEndpointURL_AllowsValidPublicURLs(t *testing.T) {
	v := NewDefaultSSRFValidator()

	cases := []struct {
		name string
		url  string
	}{
		{"https public", "https://api.example.com/v1"},
		{"http public", "http://api.example.com:8080/v1"},
		{"public IP", "http://203.0.113.1/api"},
		{"public IP with port", "https://198.51.100.50:443/api"},
		{"subdomain", "https://a1-mediator.oran-platform.example.com/A1-P/v2"},
		{"with path and query", "https://api.example.com/v1/policies?limit=10"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateEndpointURL(tc.url)
			if err != nil {
				t.Errorf("expected nil error for %q, got: %v", tc.url, err)
			}
		})
	}
}

func TestValidateEndpointURL_AllowsHostnamesNotResolvable(t *testing.T) {
	v := NewDefaultSSRFValidator()

	cases := []struct {
		name string
		url  string
	}{
		{"internal service name", "http://porch-server.nephoran-system.svc.cluster.local:8080/api"},
		{"short service name", "http://weaviate:8080/v1/objects"},
		{"custom domain", "https://my-internal-api.company.io/v1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateEndpointURL(tc.url)
			if err != nil {
				t.Errorf("expected nil for %q, got: %v", tc.url, err)
			}
		})
	}
}

func TestSSRFValidatorConfig_AllowLocalhost(t *testing.T) {
	v := NewSSRFValidator(SSRFValidatorConfig{
		AllowLocalhost: true,
	})

	cases := []struct {
		name      string
		url       string
		expectErr bool
	}{
		{"localhost allowed", "http://localhost:8080/api", false},
		{"127.0.0.1 allowed", "http://127.0.0.1/api", false},
		{"::1 allowed", "http://[::1]/api", false},
		{"private IP still blocked", "http://10.0.0.1/api", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateEndpointURL(tc.url)
			if tc.expectErr && err == nil {
				t.Errorf("expected error for %q, got nil", tc.url)
			}
			if !tc.expectErr && err != nil {
				t.Errorf("expected nil for %q, got: %v", tc.url, err)
			}
		})
	}
}

func TestSSRFValidatorConfig_AllowPrivateIPs(t *testing.T) {
	v := NewSSRFValidator(SSRFValidatorConfig{
		AllowPrivateIPs: true,
	})

	cases := []struct {
		name      string
		url       string
		expectErr bool
	}{
		{"10.x allowed", "http://10.0.0.1/api", false},
		{"172.16.x allowed", "http://172.16.0.1/api", false},
		{"192.168.x allowed", "http://192.168.1.1/api", false},
		{"localhost still blocked", "http://localhost/api", true},
		{"metadata still blocked", "http://169.254.169.254/meta", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateEndpointURL(tc.url)
			if tc.expectErr && err == nil {
				t.Errorf("expected error for %q, got nil", tc.url)
			}
			if !tc.expectErr && err != nil {
				t.Errorf("expected nil for %q, got: %v", tc.url, err)
			}
		})
	}
}

func TestSSRFValidatorConfig_AllowedHosts(t *testing.T) {
	v := NewSSRFValidator(SSRFValidatorConfig{
		AllowedHosts: []string{
			"porch-server",
			"weaviate.weaviate.svc.cluster.local",
		},
	})

	cases := []struct {
		name      string
		url       string
		expectErr bool
	}{
		{"allowed host porch-server", "http://porch-server:8080/api", false},
		{"allowed host weaviate", "http://weaviate.weaviate.svc.cluster.local:8080/v1", false},
		{"not-allowed host", "http://evil-server:8080/api", false}, // hostname, not IP, passes
		{"metadata still blocked", "http://169.254.169.254/meta", true},
		{"localhost still blocked", "http://localhost/api", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateEndpointURL(tc.url)
			if tc.expectErr && err == nil {
				t.Errorf("expected error for %q, got nil", tc.url)
			}
			if !tc.expectErr && err != nil {
				t.Errorf("expected nil for %q, got: %v", tc.url, err)
			}
		})
	}
}

func TestSSRFError_Format(t *testing.T) {
	err := &SSRFError{URL: "file:///etc/passwd", Reason: "scheme not allowed"}
	expected := `SSRF protection: URL "file:///etc/passwd" rejected: scheme not allowed`
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestValidateEndpointURL_ConvenienceFunction(t *testing.T) {
	if err := ValidateEndpointURL("https://api.example.com/v1"); err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
	if err := ValidateEndpointURL("file:///etc/passwd"); err == nil {
		t.Error("expected error for file:// scheme, got nil")
	}
}

func TestValidateEndpointURLWithAllowlist(t *testing.T) {
	err := ValidateEndpointURLWithAllowlist(
		"http://porch-server:8080/api",
		[]string{"porch-server"},
	)
	if err != nil {
		t.Errorf("expected nil for allowed host, got: %v", err)
	}
}

func TestValidateInClusterEndpointURL(t *testing.T) {
	if err := ValidateInClusterEndpointURL("http://10.96.0.1:443/api"); err != nil {
		t.Errorf("expected nil for in-cluster IP, got: %v", err)
	}
	if err := ValidateInClusterEndpointURL("http://169.254.169.254/meta"); err == nil {
		t.Error("expected error for metadata endpoint, got nil")
	}
	if err := ValidateInClusterEndpointURL("file:///etc/passwd"); err == nil {
		t.Error("expected error for file:// scheme, got nil")
	}
}

func TestIsPrivateIP(t *testing.T) {
	cases := []struct {
		ip      string
		private bool
	}{
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"172.15.255.255", false},
		{"172.32.0.0", false},
		{"192.168.0.1", true},
		{"192.168.255.255", true},
		{"8.8.8.8", false},
		{"203.0.113.1", false},
	}

	for _, tc := range cases {
		t.Run(tc.ip, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %q", tc.ip)
			}
			got := isPrivateIP(ip)
			if got != tc.private {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tc.ip, got, tc.private)
			}
		})
	}
}

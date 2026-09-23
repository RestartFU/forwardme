package config

type Config struct {
	Routes                 map[string]string
	TCPRoutes              map[string]string
	UDPRoutes              map[string]string
	Email                  string
	CertificatesPath       string
	CloudflareAPITokenFile string
}

func DefaultConfig() Config {
	return Config{
		Routes:                 map[string]string{"example.com": "http://api.example.com:3456"},
		TCPRoutes:              map[string]string{},
		UDPRoutes:              map[string]string{},
		Email:                  "contact@example.com",
		CertificatesPath:       "/var/lib/certmagic",
		CloudflareAPITokenFile: "",
	}
}

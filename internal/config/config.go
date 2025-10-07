package config

type Config struct {
	Routes           map[string]string
	Email            string
	CertificatesPath string
}

func DefaultConfig() Config {
	return Config{
		Routes:           map[string]string{"example.com": "http://api.example.com:3456"},
		Email:            "contact@example.com",
		CertificatesPath: "/var/lib/certmagic",
	}
}

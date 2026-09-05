package main

import (
	"fmt"
	"log"
	"maps"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/caddyserver/certmagic"
	"github.com/libdns/cloudflare"
	"github.com/restartfu/forwardme/internal/config"
	"github.com/restartfu/gophig"
)

func main() {
	cfg, err := loadConfig("config.toml")
	if err != nil {
		log.Fatalln(err)
	}

	routes := cfg.Routes
	domains := slices.Collect(maps.Keys(routes))

	handler := http.HandlerFunc(handleFunc(routes))

	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.Email = cfg.Email
	certmagic.Default.Storage = &certmagic.FileStorage{Path: cfg.CertificatesPath}

	log.Println("Starting HTTPS reverse proxy with automatic Let's Encrypt...")
	log.Println("Domains:", domains)
	log.Fatal(certmagic.HTTPS(domains, handler))
}

func loadConfig(configPath string) (config.Config, error) {
	defaultConfig := config.DefaultConfig()
	g := gophig.NewGophig[config.Config](configPath, gophig.TOMLMarshaler{}, os.ModePerm)
	conf, err := g.LoadConf()
	if err != nil {
		if os.IsNotExist(err) {
			err = g.SaveConf(defaultConfig)
			return defaultConfig, fmt.Errorf("could not save default config: %w", err)
		}
		return config.Config{}, fmt.Errorf("could not load config: %w", err)
	}
	if conf.CloudflareAPITokenFile != "" {
		tokenBytes, err := os.ReadFile(conf.CloudflareAPITokenFile)
		if err != nil {
			return config.Config{}, fmt.Errorf("could not read Cloudflare API token file: %w", err)
		}
		token := strings.TrimSpace(string(tokenBytes))
		if token == "" {
			return config.Config{}, fmt.Errorf("Cloudflare API token file is empty")
		}
		certmagic.DefaultACME.DNS01Solver = &certmagic.DNS01Solver{
			DNSManager: certmagic.DNSManager{
				DNSProvider: &cloudflare.Provider{APIToken: token},
			},
		}
	}
	return conf, nil
}

func handleFunc(routes map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host := strings.Split(r.Host, ":")[0]

		targetURL, exists := routes[host]
		if !exists {
			http.Error(w, "Domain not found", http.StatusNotFound)
			log.Printf("Unknown domain: %s", host)
			return
		}

		target, err := url.Parse(targetURL)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			log.Printf("Error parsing target URL: %v", err)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(target)

		proxy.Director = func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host
			req.Header.Set("X-Forwarded-Host", r.Host)
			req.Header.Set("X-Forwarded-Proto", "https")
			req.Header.Set("X-Real-IP", strings.Split(r.RemoteAddr, ":")[0])
		}

		log.Printf("Forwarding %s -> %s", host, targetURL)
		proxy.ServeHTTP(w, r)
	}
}

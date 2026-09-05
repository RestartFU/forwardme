package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/caddyserver/certmagic"
)

func TestLoadConfigEnablesDNS01FromTokenFile(t *testing.T) {
	oldSolver := certmagic.DefaultACME.DNS01Solver
	t.Cleanup(func() { certmagic.DefaultACME.DNS01Solver = oldSolver })
	certmagic.DefaultACME.DNS01Solver = nil

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "cloudflare-token")
	if err := os.WriteFile(tokenPath, []byte("secret-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.toml")
	config := "Email = 'contact@example.com'\n" +
		"CertificatesPath = 'certmagic'\n" +
		"CloudflareAPITokenFile = '" + tokenPath + "'\n\n" +
		"[Routes]\n"
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := loadConfig(configPath); err != nil {
		t.Fatalf("loadConfig returned an error: %v", err)
	}
	if certmagic.DefaultACME.DNS01Solver == nil {
		t.Fatal("expected Cloudflare DNS-01 solver to be configured")
	}
}

func TestLoadConfigRejectsMissingCloudflareTokenFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	config := "Email = 'contact@example.com'\n" +
		"CertificatesPath = 'certmagic'\n" +
		"CloudflareAPITokenFile = '" + filepath.Join(dir, "missing-token") + "'\n\n" +
		"[Routes]\n"
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := loadConfig(configPath); err == nil {
		t.Fatal("expected missing Cloudflare token file to be rejected")
	}
}

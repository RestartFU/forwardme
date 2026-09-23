# forwardme

Forwardme serves HTTPS routes by domain and can also forward raw TCP traffic by listening address. TCP traffic is routed by port.

```toml
Email = 'contact@example.com'
CertificatesPath = '/var/lib/forwardme/certmagic'

[Routes]
'example.com' = 'http://app:8080'

[TCPRoutes]
':25565' = 'minecraft-java:25565'

```

The keys in `TCPRoutes` are local `host:port` listening addresses. Use `:port` to listen on all interfaces or `127.0.0.1:port` for local access. Values are backend `host:port` addresses; use brackets around IPv6 addresses, such as `[::1]:25565`. TCP connections are streamed in both directions.

When running in Docker, publish every configured port with the matching protocol in Compose, for example:

```yaml
ports:
  - "80:80"
  - "443:443"
  - "25565:25565/tcp"
```

No TCP ports beyond HTTPS are opened unless you add routes and publish those ports.

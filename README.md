# forwardme

Forwardme serves HTTPS routes by domain and can also forward raw TCP and UDP traffic by listening address. TCP and UDP traffic is routed by port; those protocols do not carry an HTTP host name for domain routing.

```toml
Email = 'contact@example.com'
CertificatesPath = '/var/lib/forwardme/certmagic'

[Routes]
'example.com' = 'http://app:8080'

[TCPRoutes]
':25565' = 'minecraft-java:25565'

[UDPRoutes]
':19132' = 'minecraft-bedrock:19132'
```

The keys in `TCPRoutes` and `UDPRoutes` are local `host:port` listening addresses. Use `:port` to listen on all interfaces or `127.0.0.1:port` for local access. Values are backend `host:port` addresses; use brackets around IPv6 addresses, such as `[::1]:19132`. TCP connections are streamed in both directions. UDP replies return to the originating client, and idle UDP sessions expire after one minute.

When running in Docker, publish every configured port with the matching protocol in Compose, for example:

```yaml
ports:
  - "80:80"
  - "443:443"
  - "25565:25565/tcp"
  - "19132:19132/udp"
```

No TCP or UDP ports beyond HTTPS are opened unless you add routes and publish those ports.

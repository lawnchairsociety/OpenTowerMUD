# Deployment Guide

This guide covers deploying OpenTowerMUD with a reverse proxy for secure WebSocket connections.

## Server Ports

| Protocol | Default Port | Description |
|----------|--------------|-------------|
| Telnet   | 4000         | Traditional MUD client access |
| WebSocket| 4443         | Browser/web client access |

## Reverse Proxy Setup (nginx)

Using a reverse proxy is recommended for production to handle TLS/SSL termination.

### 1. Install nginx

```bash
# Ubuntu/Debian
sudo apt install nginx

# CentOS/RHEL
sudo yum install nginx
```

### 2. Obtain SSL Certificates

Use Let's Encrypt for free certificates:

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d yourdomain.com
```

### 3. Configure nginx

Copy the example config:

```bash
sudo cp nginx.conf /etc/nginx/sites-available/opentowermud
sudo ln -s /etc/nginx/sites-available/opentowermud /etc/nginx/sites-enabled/
```

Edit the config to update:
- `server_name` - your domain
- SSL certificate paths (certbot usually handles this)

### 4. Test and Reload

```bash
sudo nginx -t
sudo systemctl reload nginx
```

### 5. Connect

Web clients can now connect via:
```
wss://yourdomain.com/ws
```

## Alternative: Caddy

Caddy provides automatic HTTPS with minimal configuration:

```
yourdomain.com {
    reverse_proxy /ws* localhost:4443
}
```

## Firewall

Ensure these ports are open:
- 80 (HTTP redirect)
- 443 (HTTPS/WSS)
- 4000 (Telnet, if allowing direct access)

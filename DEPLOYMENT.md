# Deployment Guide

## Docker Deployment

### Prerequisites
- Docker and Docker Compose installed
- GitHub account (for CI/CD)

### Local Development with Docker

1. **Build the image:**
   ```bash
   docker build -t small-rpg-adhd-monolith .
   ```

2. **Run with Docker Compose:**
   ```bash
   # Create .env file with your secrets (optional)
   echo "SESSION_SECRET=your-secret-key" > .env
   echo "TELEGRAM_BOT_TOKEN=your-bot-token" >> .env
   
   # Start the application
   docker-compose up -d
   ```

3. **View logs:**
   ```bash
   docker-compose logs -f app
   ```

4. **Stop the application:**
   ```bash
   docker-compose down
   ```

### Environment Variables

- `PORT` - Server port (default: 8080)
- `SESSION_SECRET` - Secret key for session encryption (required in production)
- `TELEGRAM_BOT_TOKEN` - Telegram bot token (optional)
- `DB_PATH` - Database file path (default: /app/data/small-rpg.db)

### CI/CD with GitHub Actions

The repository includes a GitHub Actions workflow that automatically builds and publishes Docker images to GitHub Container Registry (ghcr.io).

#### Setup

1. **Enable GitHub Actions:**
   - Go to your repository settings
   - Navigate to Actions → General
   - Ensure actions are enabled

2. **Configure Package Visibility:**
   - Go to your repository settings
   - Navigate to Packages
   - Set visibility to Public or Private as needed

3. **Trigger Builds:**
   - Push to `main` branch → builds `latest` tag
   - Push a tag like `v1.0.0` → builds version tags

#### Image Tags

The workflow automatically creates the following tags:
- `latest` - Latest build from main branch
- `main` - Latest build from main branch
- `v1.2.3` - Semantic version tags
- `v1.2` - Minor version tags
- `v1` - Major version tags
- `main-<sha>` - Commit SHA tags

### Authenticating with GitHub Container Registry

**Important:** By default, packages published to GitHub Container Registry (GHCR) are **Private**. If you see an error like `Error response from daemon: error from registry: denied`, you need to authenticate before pulling the image.

#### Option 1: Authenticate with Docker (For Private Packages)

To pull private images from GHCR, you need a GitHub Personal Access Token (PAT) with `read:packages` scope.

1. **Create a Personal Access Token:**
   - Go to GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
   - Click "Generate new token (classic)"
   - Give it a descriptive name (e.g., "Docker GHCR Access")
   - Select the following scope:
     - `read:packages` - Download packages from GitHub Package Registry
   - Click "Generate token" and **copy the token** (you won't see it again)

2. **Authenticate Docker with GHCR:**
   ```bash
   # Login to GHCR using your GitHub username and PAT
   echo "YOUR_PAT_TOKEN" | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
   ```

   Example:
   ```bash
   echo "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" | docker login ghcr.io -u supercakecrumb --password-stdin
   ```

3. **Verify authentication:**
   ```bash
   docker pull ghcr.io/supercakecrumb/small-rpg-adhd-monolith:latest
   ```

4. **Save credentials (optional):**
   Docker will save your credentials in `~/.docker/config.json`. You only need to login once per machine.

#### Option 2: Make the Package Public (No Authentication Required)

If you want anyone to pull your image without authentication, you can change the package visibility to Public. This allows anyone to pull the container image without needing a GitHub Personal Access Token or Docker login.

**Step-by-step instructions to make the package public:**

1. **Go to your GitHub repository main page**
   - Navigate to `https://github.com/YOUR_USERNAME/small-rpg-adhd-monolith`

2. **Locate the Packages section**
   - Look for the "Packages" section on the right sidebar
   - Alternatively, click the "Packages" tab if it's visible in the repository navigation

3. **Click on the package name**
   - Click on `small-rpg-adhd-monolith` in the packages list

4. **Open Package settings**
   - On the package page, look for "Package settings" in the right sidebar
   - Click "Package settings"

5. **Navigate to the Danger Zone**
   - Scroll down to the bottom of the settings page
   - Find the "Danger Zone" section

6. **Change visibility**
   - Click "Change visibility"
   - A dialog will appear asking you to confirm

7. **Select Public and confirm**
   - Select "Public" from the options
   - Type the package name to confirm
   - Click the confirmation button

**Result:** After making the package public, anyone can pull it without authentication:
```bash
docker pull ghcr.io/supercakecrumb/small-rpg-adhd-monolith:latest
```

No GitHub Personal Access Token or `docker login` is required for public packages.

#### Docker Compose with Private Images

If using a private image in [`docker-compose.yml`](docker-compose.yml:1), you must authenticate first:

```bash
# 1. Login to GHCR
echo "YOUR_PAT_TOKEN" | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin

# 2. Update docker-compose.yml to use the GHCR image
# Change the 'build' section to 'image':
# services:
#   app:
#     image: ghcr.io/supercakecrumb/small-rpg-adhd-monolith:latest

# 3. Pull and run
docker-compose pull
docker-compose up -d
```

### Pulling from GitHub Container Registry

```bash
# Pull the latest image
docker pull ghcr.io/supercakecrumb/small-rpg-adhd-monolith:latest

# Run the image
docker run -d \
  -p 8080:8080 \
  -e SESSION_SECRET=your-secret \
  -e TELEGRAM_BOT_TOKEN=your-token \
  -v $(pwd)/data:/app/data \
  ghcr.io/supercakecrumb/small-rpg-adhd-monolith:latest
```

### Production Deployment

#### Using Docker Compose

1. **Create production docker-compose.yml:**
   ```yaml
   version: '3.8'
   
   services:
     app:
       image: ghcr.io/supercakecrumb/small-rpg-adhd-monolith:latest
       ports:
         - "8080:8080"
       environment:
         - PORT=8080
         - SESSION_SECRET=${SESSION_SECRET}
         - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
       volumes:
         - ./data:/app/data
       restart: always
   ```

2. **Create .env file with production secrets:**
   ```bash
   SESSION_SECRET=<generate-strong-random-key>
   TELEGRAM_BOT_TOKEN=<your-bot-token>
   ```

3. **Deploy:**
   ```bash
   docker-compose up -d
   ```

#### Database Persistence

The SQLite database is stored in `/app/data/small-rpg.db` inside the container. Make sure to:
- Mount a volume to persist data across container restarts
- Regularly backup the `data/` directory
- Consider using a backup solution for production

#### Security Best Practices

1. **Generate a strong SESSION_SECRET:**
   ```bash
   openssl rand -base64 32
   ```

2. **Use secrets management:**
   - Docker secrets
   - Kubernetes secrets
   - Cloud provider secret managers

3. **Enable HTTPS:**
   - Use a reverse proxy (nginx, Traefik, Caddy)
   - Configure TLS certificates (Let's Encrypt recommended)

4. **Network security:**
   - Use Docker networks to isolate containers
   - Configure firewall rules
   - Limit exposed ports

### Health Checks

The Docker Compose configuration includes health checks:
- Endpoint: `http://localhost:8080/`
- Interval: 30 seconds
- Timeout: 10 seconds
- Retries: 3

### Troubleshooting

1. **Container won't start:**
   ```bash
   docker-compose logs app
   ```

2. **Check container status:**
   ```bash
   docker-compose ps
   ```

3. **Access container shell:**
   ```bash
   docker-compose exec app sh
   ```

4. **Database issues:**
   - Check volume mount permissions
   - Ensure `/app/data` directory is writable
   - Verify SQLite database file exists

### Monitoring

For production deployments, consider:
- Container monitoring (Prometheus, Grafana)
- Log aggregation (ELK Stack, Loki)
- Uptime monitoring (UptimeRobot, Pingdom)
- Error tracking (Sentry)

### Rollback

To rollback to a previous version:
```bash
# Pull specific version
docker pull ghcr.io/supercakecrumb/small-rpg-adhd-monolith:v1.0.0

# Update docker-compose.yml to use specific tag
# Then restart
docker-compose up -d
```

### Resource Limits

For production, set resource limits in docker-compose.yml:
```yaml
services:
  app:
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
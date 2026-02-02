# GitHub Actions Deployment

Automated deployment system for the Jobber application using GitHub Actions, Docker Hub, and Docker Compose.

## 📚 Documentation

| File | Purpose |
|------|---------|
| **[QUICK_REFERENCE.md](QUICK_REFERENCE.md)** | Quick commands and cheat sheet (START HERE!) |
| **[DEPLOYMENT_SETUP.md](DEPLOYMENT_SETUP.md)** | Complete setup guide with troubleshooting |
| **[workflows/deploy-dev.yml](workflows/deploy-dev.yml)** | The actual workflow (extensively commented) |
| **[server-setup.sh](server-setup.sh)** | Server preparation script |

## 🚀 Quick Start

### 1. Prepare Server (One-Time)
```bash
# Copy setup script to your server
scp .github/server-setup.sh user@your-server-ip:~/

# SSH and run it
ssh user@your-server-ip
chmod +x server-setup.sh
./server-setup.sh
```

### 2. Configure GitHub Secrets
Add these secrets to your GitHub repository (Settings → Secrets → Actions):
- `DOCKERHUB_USERNAME` - Your Docker Hub username
- `DOCKERHUB_TOKEN` - Docker Hub access token
- `SSH_PRIVATE_KEY` - SSH private key for server access
- `SERVER_HOST` - Server IP address
- `SERVER_USER` - SSH username

**See [DEPLOYMENT_SETUP.md](DEPLOYMENT_SETUP.md) for detailed instructions.**

### 3. Deploy
```bash
git push origin dev  # Triggers automatic deployment
```

## ✨ Features

- ✅ **Path-based deployment** - Only builds changed services
- ✅ **Immutable tagging** - Every image tagged with commit SHA
- ✅ **Zero-downtime deploys** - Rolling container restart
- ✅ **Data persistence** - Named volumes survive deployments
- ✅ **Secret safety** - Never touches `.env` file on server
- ✅ **Easy rollbacks** - Re-run workflow on any commit

## 🔐 Security Model

**Runtime Secrets (stored in `.env` on server):**
- Database credentials
- JWT secrets
- S3/Object Storage credentials
- API keys

**CI/CD Secrets (stored in GitHub Secrets):**
- Docker Hub authentication
- SSH private key for deployment
- Server connection details

**Key principle:** GitHub Actions NEVER creates or modifies the `.env` file. It contains sensitive runtime secrets that must be manually provisioned on the server.

## 📖 Learn More

- **New to this setup?** → Read [QUICK_REFERENCE.md](QUICK_REFERENCE.md)
- **First deployment?** → Follow [DEPLOYMENT_SETUP.md](DEPLOYMENT_SETUP.md)
- **Need help?** → Check troubleshooting section in [DEPLOYMENT_SETUP.md](DEPLOYMENT_SETUP.md)
- **Want to understand the workflow?** → Read comments in [workflows/deploy-dev.yml](workflows/deploy-dev.yml)

## 🎯 Deployment Flow

```
Developer Push
      ↓
GitHub Actions
      ↓
  Detect Changes (path filters)
      ↓
  ┌─────────────┬─────────────┐
  ↓             ↓             ↓
Backend       Frontend      Skip
Build         Build         (no changes)
  ↓             ↓
Push to       Push to
Docker Hub    Docker Hub
  └─────────────┴─────────────┘
              ↓
         SSH to Server
              ↓
      docker compose pull
              ↓
      docker compose up -d
              ↓
       ✅ Deployed!
```

## 📞 Support

Questions? Check the documentation files above or review the workflow comments.

**Pro tip:** Start with [QUICK_REFERENCE.md](QUICK_REFERENCE.md) for the most common commands and workflows.

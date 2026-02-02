# Terraform Infrastructure for Jobber DEV

This directory contains Terraform configuration for provisioning a Hetzner Cloud server for the Jobber application.

## Quick Start

```bash
# 1. Copy example variables
cp terraform.tfvars.example terraform.tfvars

# 2. Edit with your values
nano terraform.tfvars

# 3. Initialize Terraform
terraform init

# 4. Preview changes
terraform plan

# 5. Create infrastructure
terraform apply
```

## Files

- `main.tf` - Main infrastructure definition (server, firewall, SSH key)
- `variables.tf` - Input variables
- `outputs.tf` - Output values (IP, SSH command)
- `cloud-init.yaml` - Server provisioning script (Docker installation)
- `terraform.tfvars.example` - Example configuration (copy to `terraform.tfvars`)

## What Gets Created

- **1x Hetzner Cloud Server**
  - Type: CX22 (2 vCPU, 4GB RAM, 40GB disk)
  - OS: Ubuntu 22.04 LTS
  - Cost: ~€5.83/month

- **1x SSH Key** (imported from your local machine)

- **1x Firewall**
  - Port 22 (SSH)
  - Port 80 (HTTP)
  - Port 443 (HTTPS)

## Server Provisioning

The server is automatically configured with:
- Docker Engine (latest)
- Docker Compose Plugin
- Application directory (`/opt/jobber`)

This is handled by `cloud-init.yaml` and runs on first boot.

## Important Notes

- The server will NOT be recreated on subsequent `terraform apply` calls
- Cloud-init runs once on first boot (takes 2-3 minutes)
- SSH key and user_data changes are ignored after creation
- Destroying infrastructure does NOT affect external Object Storage

## Useful Commands

```bash
# Get server IP
terraform output server_ip

# SSH to server
terraform output -raw ssh_command | sh

# Or manually
ssh root@$(terraform output -raw server_ip)

# Destroy everything
terraform destroy
```

## Cost Optimization

Current setup uses CX22 (~€5.83/month). For even cheaper:

- **CX11** (1 vCPU, 2GB RAM) - €3.29/month
  - May struggle with multiple containers
  - Not recommended for running all services

- **CAX11** (ARM, 2 vCPU, 4GB RAM) - €3.81/month
  - Requires ARM-compatible Docker images
  - Your Go backend is compatible (builds for ARM)
  - Check if all your dependencies support ARM

To change server type:
```hcl
# In terraform.tfvars
server_type = "cax11"  # ARM-based, cheaper
```

## Security

Current configuration:
- ✅ Firewall enabled
- ✅ Only necessary ports exposed
- ✅ SSH key authentication only (no password)
- ❌ No automatic updates configured
- ❌ No monitoring/alerting

For production, consider:
- Restricting SSH to specific IPs
- Adding fail2ban
- Setting up monitoring (Prometheus/Grafana)
- Implementing automated backups

## Troubleshooting

### "Authentication failed" when applying
- Check your `hcloud_token` in `terraform.tfvars`
- Verify token is active in Hetzner Console

### "SSH key already exists"
- If you've run this before, Terraform may think resource exists
- Option 1: Import existing: `terraform import hcloud_ssh_key.default <key_id>`
- Option 2: Change `project_name` variable to create new resources

### Docker not installed after creation
- Cloud-init takes 2-3 minutes to complete
- SSH to server and check: `tail -f /var/log/cloud-init-output.log`
- If stuck, manually run: `cloud-init status --wait`

### Can't SSH to server
- Check firewall allows your IP (current config allows all)
- Verify SSH key matches: `ssh-add -l`
- Try verbose mode: `ssh -v root@<ip>`

## State Management

Terraform state is stored locally in `terraform.tfstate`.

**⚠️ Important:**
- This file contains sensitive data (passwords, tokens)
- Never commit `terraform.tfstate` to git (already in `.gitignore`)
- For team environments, use remote state (Terraform Cloud, S3, etc.)

## Next Steps After Infrastructure Creation

1. Note the server IP from `terraform output`
2. Wait 2-3 minutes for cloud-init to complete
3. SSH to server: `ssh root@<ip>`
4. Verify Docker is installed: `docker --version`
5. Follow deployment steps in `/DEPLOYMENT.md`

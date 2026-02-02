terraform {
  required_version = ">= 1.0"
  
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.45"
    }
  }
}

provider "hcloud" {
  token = var.hcloud_token
}

# SSH Key - use existing key by name
# If you need to create a new key, use a different SSH public key
data "hcloud_ssh_key" "default" {
  name = "gha-deploy"  # Use your existing SSH key
}

# Firewall rules - only HTTP, HTTPS, and SSH
resource "hcloud_firewall" "web" {
  name = "${var.project_name}-firewall-${formatdate("YYYYMMDDHHmmss", timestamp())}"
  
  lifecycle {
    ignore_changes = [name]
  }

  # SSH access
  rule {
    direction = "in"
    protocol  = "tcp"
    port      = "22"
    source_ips = [
      "0.0.0.0/0",
      "::/0"
    ]
  }

  # HTTP
  rule {
    direction = "in"
    protocol  = "tcp"
    port      = "80"
    source_ips = [
      "0.0.0.0/0",
      "::/0"
    ]
  }

  # HTTPS (placeholder for future SSL setup)
  rule {
    direction = "in"
    protocol  = "tcp"
    port      = "443"
    source_ips = [
      "0.0.0.0/0",
      "::/0"
    ]
  }
}

# Main server - cheapest instance with Ubuntu LTS
resource "hcloud_server" "app" {
  name        = "${var.project_name}-dev"
  server_type = var.server_type
  image       = var.server_image
  location    = var.server_location
  ssh_keys    = [data.hcloud_ssh_key.default.id]
  firewall_ids = [hcloud_firewall.web.id]

  # Cloud-init for Docker installation
  user_data = file("${path.module}/cloud-init.yaml")

  # Prevent recreation on every apply
  lifecycle {
    ignore_changes = [
      user_data,  # Don't recreate if cloud-init changes
      ssh_keys    # Don't recreate if SSH keys change
    ]
  }

  # Labels for organization
  labels = {
    environment = "dev"
    project     = var.project_name
    managed_by  = "terraform"
  }
}

variable "hcloud_token" {
  description = "Hetzner Cloud API Token"
  type        = string
  sensitive   = true
}

variable "ssh_public_key" {
  description = "SSH public key for server access"
  type        = string
}

variable "project_name" {
  description = "Project name used for resource naming"
  type        = string
  default     = "jobber"
}

variable "server_type" {
  description = "Hetzner server type (cpx22 = 2vCPU, 4GB RAM, 80GB disk)"
  type        = string
  default     = "cpx22"  # €6.99/month - smallest viable option for running multiple containers
}

variable "server_image" {
  description = "Server OS image"
  type        = string
  default     = "ubuntu-22.04"  # Ubuntu 22.04 LTS
}

variable "server_location" {
  description = "Server location/datacenter"
  type        = string
  default     = "nbg1"  # Nuremberg, Germany - matches your S3 region
}

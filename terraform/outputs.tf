output "server_ip" {
  description = "Public IP address of the server"
  value       = hcloud_server.app.ipv4_address
}

output "server_name" {
  description = "Server hostname"
  value       = hcloud_server.app.name
}

output "ssh_command" {
  description = "SSH command to connect to the server"
  value       = "ssh root@${hcloud_server.app.ipv4_address}"
}

output "server_status" {
  description = "Server status"
  value       = hcloud_server.app.status
}

output "server_location" {
  description = "Server location"
  value       = hcloud_server.app.location
}

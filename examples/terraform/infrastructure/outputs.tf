output "grafana_url" {
  value = "http://localhost:${var.grafana_port}"
}

output "grafana_credentials" {
  value = {
    username = var.grafana_admin_user
    password = var.grafana_admin_password
  }
  sensitive = true
}

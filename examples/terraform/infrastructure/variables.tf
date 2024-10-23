variable "grafana_port" {
  description = "Port to expose Grafana on localhost"
  type        = number
  default     = 3000
}

variable "grafana_version" {
  description = "Grafana Docker image version"
  type        = string
  default     = "11.2.2"
}

variable "grafana_admin_user" {
  description = "Grafana admin username"
  type        = string
  default     = "admin"
}

variable "grafana_admin_password" {
  description = "Grafana admin password"
  type        = string
  default     = "admin"
  sensitive   = true
}

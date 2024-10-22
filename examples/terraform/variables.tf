variable "grafana_url" {
  description = "URL of the Grafana instance"
  type        = string
  default     = "http://localhost:3000"
}

variable "grafana_auth" {
  description = "Basic auth credentials for Grafana"
  type        = string
  default     = "admin:admin"
  sensitive   = true
}

terraform {
  required_providers {
    grafana = {
      source  = "grafana/grafana"
      version = "~> 2.8.0"
    }
  }
  required_version = ">= 1.0.0"
}

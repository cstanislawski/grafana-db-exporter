output "dashboard_count" {
  value = length(grafana_dashboard.managed_dashboards)
}

output "grafana_folders" {
  value = [for folder in grafana_folder.managed_folders : folder.title]
}

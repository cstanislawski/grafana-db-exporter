provider "grafana" {
  url  = var.grafana_url
  auth = var.grafana_auth
}

resource "grafana_folder" "managed_folders" {
  for_each = toset([
    for f in fileset(path.module, "dashboards/**/")
    : dirname(f) if length(split("/", f)) > 2
  ])

  title = replace(basename(each.key), "-", " ")
}

resource "grafana_dashboard" "managed_dashboards" {
  for_each = fileset(path.module, "dashboards/**/*.json")

  config_json = file("${path.module}/${each.value}")

  folder = length(split("/", each.value)) > 2 ? (
    grafana_folder.managed_folders[dirname(each.value)].id
  ) : 0

  overwrite = true
}

provider "docker" {}

resource "docker_network" "grafana_network" {
  name = "grafana_network"
}

resource "docker_volume" "grafana_data" {
  name = "grafana_data"
}

resource "docker_container" "grafana" {
  name  = "grafana"
  image = "grafana/grafana:${var.grafana_version}"

  ports {
    internal = 3000
    external = var.grafana_port
    ip       = "127.0.0.1"
  }

  volumes {
    volume_name    = docker_volume.grafana_data.name
    container_path = "/var/lib/grafana"
  }

  networks_advanced {
    name = docker_network.grafana_network.name
  }

  env = [
    "GF_SECURITY_ADMIN_USER=${var.grafana_admin_user}",
    "GF_SECURITY_ADMIN_PASSWORD=${var.grafana_admin_password}",
    "GF_SERVER_HTTP_ADDR=0.0.0.0",
    "GF_SERVER_HTTP_PORT=${var.grafana_port}",
    "GF_AUTH_ANONYMOUS_ENABLED=false",
    "GF_USERS_ALLOW_SIGN_UP=false"
  ]

  restart = "unless-stopped"
}

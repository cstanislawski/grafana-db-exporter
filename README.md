# grafana-db-exporter

Export your ClickOps'ed Grafana dashboards into the repository.

## v1.0 requirements

- List all dashboards within an organization ORG_ID,
- export dashboards as json,
- upload dashboards to SSH_URL git repository and create a PR based on BASE_BRANCH, to a path BASE_PATH

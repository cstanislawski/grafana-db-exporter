# grafana-db-exporter

Export your ClickOps'ed Grafana dashboards into the repository.

## v1.0 requirements

List all dashboards within an organization ORG_ID, export it as json, upload it to SSH_URL git repo with base of BASE_BRANCH, to a path BASE_PATH.

## todo - features

- multiple orgs,
- multiple repos,
- custom branch name generation,
- multiple paths within the repo based on the org id / parameter / etc,
- export as k8s ConfigMap / Secret

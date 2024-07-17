# grafana-db-exporter

Export your ClickOps'ed Grafana dashboards into the repository.

## v1.0 requirements

- List all dashboards within an organization ORG_ID,
- export dashboards as json,
- upload dashboards to SSH_URL git repository and create a PR based on BASE_BRANCH, to a path BASE_PATH,
- push the branch to the repository only if there are changes.

## Usage

`grafana-db-exporter` is a utility tool meant to be utilized either as a periodic job on a CI/CD pipeline, or as a Kubernetes CronJob.

### Configuration / Environment variables

Required:

- `SSH_URL`: SSH URL of the repository to push the dashboards to,
- `SSH_KEY`: Path to the SSH key to use to authenticate with the repository,
- `SSH_USER`: SSH user to use to authenticate with the repository,
- `SSH_EMAIL`: SSH email to use to authenticate with the repository,
- `REPO_SAVE_PATH`: Path to save the dashboards to in the repository,
- `GRAFANA_URL`: URL of the Grafana instance to export the dashboards from,
- `GRAFANA_API_KEY`: API / Service Account key to use to authenticate with the Grafana instance

Optional:

- `BASE_BRANCH`: Branch to create the PR against. Defaults to `main`,
- `BRANCH_PREFIX`: Prefix to use for the branch name. Defaults to `grafana-db-exporter-`
- `SSH_KEY_PASSWORD`: Passphrase to use to decrypt the SSH key. Defaults to `""`,
- `SSH_ACCEPT_UNKNOWN_HOSTS`: Whether to ignore unknown hosts when connecting to the Grafana instance, defaults to `false`

Conditional:

- `SSH_KNOWN_HOSTS_PATH`: The path to the known hosts file to use when connecting to the Grafana instance, required if `SSH_ACCEPT_UNKNOWN_HOSTS` is `false`,

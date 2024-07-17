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

- `SSH_URL`: The SSH URL of the repository to push the dashboards to,
- `SSH_KEY`: The SSH key to use to authenticate with the repository,
- `SSH_USER`: The SSH user to use to authenticate with the repository,
- `SSH_EMAIL`: The SSH email to use to authenticate with the repository,
- `REPO_SAVE_PATH`: The path to save the dashboards to in the repository,
- `GRAFANA_URL`: The URL of the Grafana instance to export the dashboards from,
- `GRAFANA_API_KEY`: The API / Service Account key to use to authenticate with the Grafana instance,

Optional:

- `BASE_BRANCH`: The branch to create the PR against. Defaults to `main`,
- `BRANCH_PREFIX`: The prefix to use for the branch name. Defaults to `grafana-db-exporter-`

# grafana-db-exporter

Automatically export Grafana dashboards to Git for version control and Infrastructure as Code workflows.

## Why?

- Version control your Grafana dashboards
- Enable GitOps workflows for dashboard management
- Bridge the gap between UI-based dashboard creation and Infrastructure as Code
- Backup dashboards automatically

## Usage

`grafana-db-exporter` is a utility tool meant to be utilized either as a periodic job on a CI/CD pipeline, or as a Kubernetes CronJob.

## Configuration

All configuration is done via environment variables. Here's a complete reference grouped by functionality:

### Git Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SSH_URL` | ✓ | `""` | Git repository SSH URL (e.g. `git@github.com:org/repo.git`) |
| `SSH_KEY` | ✓ | `""` | Path to SSH private key (supports `rsa`, `ecdsa`, `ed25519`) |
| `SSH_USER` | ✓ | `""` | Git commit author username |
| `SSH_EMAIL` | ✓ | `""` | Git commit author email |
| `BASE_BRANCH` | | `main` | Branch to create new branches from |
| `BRANCH_PREFIX` | | `grafana-db-exporter-` | Prefix for new branch names |
| `SSH_KEY_PASSWORD` | | `""` | SSH key passphrase if encrypted |
| `SSH_KNOWN_HOSTS_PATH` | ✓* | `""` | Path to known_hosts file (*required if `SSH_ACCEPT_UNKNOWN_HOSTS=false`) |
| `SSH_ACCEPT_UNKNOWN_HOSTS` | | `false` | Skip host key verification |

### Grafana Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GRAFANA_URL` | ✓ | `""` | Grafana instance URL |
| `GRAFANA_SA_TOKEN` | ✓ | `""` | [Service Account token](https://grafana.com/docs/grafana/latest/administration/service-accounts/) (Viewer role is sufficient) |

### Export Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `REPO_SAVE_PATH` | `"grafana-dashboards"` | Directory path in repository to save dashboards |
| `IGNORE_FOLDER_STRUCTURE` | `false` | Flatten Grafana folder hierarchy in export |
| `DELETE_MISSING` | `true` | Remove dashboards that no longer exist in Grafana |
| `ADD_MISSING_NEWLINES` | `true` | Ensure JSON files end with newline |
| `DRY_RUN` | | `false` Commit changes but don't push |

### Runtime Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |
| `ENABLE_RETRIES` | `true` | Retry failed operations |
| `NUM_OF_RETRIES` | `3` | Maximum retry attempts |
| `RETRY_INTERVAL` | `5` | Seconds between retries |

### Operation Mode Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `RUN_MODE` | `one-time` | Operation mode (`one-time`, `periodic`) |
| `SYNC_INTERVAL` | `5m` | Interval between syncs in periodic mode (e.g., `60s`, `5m`, `24h`) |
| `BRANCH_STRATEGY` | `new-branch` | Branch creation strategy (`new-branch`, `reuse-branch`) |
| `BRANCH_TTL` | `24h` | How long to reuse a branch before creating new one (only for `reuse-branch` strategy) |

Example `.env` file with periodic sync:

```env
# Git Configuration
SSH_URL=git@github.com:org/repo.git
SSH_KEY=/app/.ssh/id_rsa
SSH_USER=ci-bot
SSH_EMAIL=ci-bot@company.com
SSH_KNOWN_HOSTS_PATH=/app/.ssh/known_hosts

# Grafana Configuration
GRAFANA_URL=https://grafana.company.com
GRAFANA_SA_TOKEN=glsa_1234567890

# Export Configuration
REPO_SAVE_PATH=dashboards
DELETE_MISSING=true

# Runtime Configuration
LOG_LEVEL=info
ENABLE_RETRIES=true

# Operation Mode Configuration
RUN_MODE=periodic
SYNC_INTERVAL=1h
BRANCH_STRATEGY=reuse-branch
BRANCH_TTL=24h
```

### Operation Modes

The application supports two operation modes:

1. **One-time Mode** (default)
   - Runs once and exits
   - Creates a new branch for each run
   - Ideal for CI/CD pipelines and Kubernetes CronJobs jobs
   - Original behavior of the tool

2. **Periodic Mode**
   - Continuously monitors and syncs dashboards
   - Can reuse branches to reduce branch proliferation
   - Primarily designed for Docker environments where running as a daemon is preferred, i.e. scenarios where CronJobs are not available/are harder to manage
   - Configurable sync intervals and branch rotation

When using `periodic` mode with `reuse-branch` strategy:

- Changes are accumulated in the same branch until `BRANCH_TTL` expires
- After `BRANCH_TTL` expires, a new branch is created for subsequent changes
- This helps reduce the number of branches and PRs while maintaining reasonable chunk sizes for reviews

### Examples

#### Docker

Set the environment variables and mount the SSH key to the container:

```bash
docker run -v ~/.ssh:/app/.ssh \
  -e GRAFANA_URL=https://grafana.example.com \
  -e GRAFANA_SA_TOKEN=your-token \
  -e SSH_URL=git@github.com:org/repo.git \
  -e SSH_KEY=/app/.ssh/id_rsa \
  ghcr.io/cstanislawski/grafana-db-exporter:latest
```

#### Kubernetes CronJob

Create a ConfigMap with the known hosts of the Grafana instance and the repository, and a CronJob that runs the exporter with the required environment variables:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: github-known-hosts
data:
  # GitHub: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/githubs-ssh-key-fingerprints
  # GitLab: https://docs.gitlab.com/ee/user/gitlab_com/index.html#ssh-host-keys-fingerprints
  # Bitbucket: https://confluence.atlassian.com/bitbucket/ssh-keys-935365775.html
  known_hosts: |
    github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl
    github.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wAdT/y6v0mKV0U2w0WZ2YB/++Tpockg=
    github.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQowgcQnjshcLrqPEiiphnt+VTTvDP6mHBL9j1aNUkY4Ue1gvwnGLVlOhGeYrnZaMgRK6+PKCUXaDbC7qtbW8gIkhL7aGCsOr/C56SJMy/BCZfxd1nWzAOxSDPgVsmerOBYfNqltV9/hWCqBywINIR+5dIg6JTJ72pcEpEjcYgXkE2YEFXV1JHnsKgbLWNlhScqb2UmyRkQyytRLtL+38TGxkxCflmO+5Z8CSSNY7GidjMIZ7Q4zMjA2n1nGrlTDkzwDCsw+wqFPGQA179cnfGWOWRVruj16z6XyvxvjJwbz0wQZ75XK5tKSb7FNyeIEs4TT4jk+S4dhPeAUC5y+bDYirYgM4GC7uEnztnZyaVWQ7B381AK4Qdrwt51ZqExKbQpTUNn+EjqoTwvqNj4kqx5QUCI0ThS/YkOxJCXmPUWZbhjpCg56i+2aB6CmK2JGhn57K5mj0MNdBXA4/WnwH6XoPWJzK5Nyu2zB3nAZp+S5hpQs+p1vN1/wsjk=
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: grafana-db-exporter
spec:
  schedule: "0 0 * * 1"
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: grafana-db-exporter
            image: ghcr.io/cstanislawski/grafana-db-exporter:latest
            volumeMounts:
            - name: github-known-hosts
              mountPath: /app/.ssh/known_hosts
              subPath: known_hosts
            env:
            - name: SSH_KEY
              valueFrom:
                secretKeyRef:
                  name: grafana-db-exporter
                  key: ssh-key
            - name: SSH_USER
              valueFrom:
                secretKeyRef:
                  name: grafana-db-exporter
                  key: ssh-user
            - name: SSH_EMAIL
              valueFrom:
                secretKeyRef:
                  name: grafana-db-exporter
                  key: ssh-email
            - name: SSH_URL
              value: "git@github.com:org/repo.git"
            - name: REPO_SAVE_PATH
              value: "grafana-dashboards"
            - name: GRAFANA_URL
              value: "https://grafana.example.com"
            - name: GRAFANA_SA_TOKEN
              valueFrom:
                secretKeyRef:
                  name: grafana-db-exporter
                  key: grafana-sa-token
            - name: BASE_BRANCH
              value: "main"
            - name: BRANCH_PREFIX
              value: "grafana-db-exporter-"
            - name: SSH_ACCEPT_UNKNOWN_HOSTS
              value: "false"
            - name: SSH_KNOWN_HOSTS_PATH
              value: "/app/.ssh/known_hosts"
            resources:
              requests:
                cpu: 100m
                memory: 128Mi
              limits:
                cpu: 500m
                memory: 512Mi
          volumes:
          - name: github-known-hosts
            configMap:
              name: github-known-hosts
              items:
              - key: known_hosts
                path: known_hosts
          restartPolicy: OnFailure
```

### Integration with CI/CD

To integrate `grafana-db-exporter` with your CI/CD pipeline, you can set up a periodic job that runs the exporter. This way, you can keep your Grafana dashboards in sync with your repository.

In order to integrate the exported dashboards to be applied to the Grafana instance, you can set up your repository to deploy the changes in several ways:

#### Terraform

You can use the [Grafana provider](https://registry.terraform.io/providers/grafana/grafana/latest/docs) and source all of your dashboards from the `REPO_SAVE_PATH` directory.

More info: [Terraform Implementation Example](examples/terraform/README.md)

#### Kubernetes

##### Basic

TBD

##### ArgoCD

TBD

### Limitations

#### Multiple Grafana organizations / instances

If you need to export dashboards from multiple Grafana organizations, you will need to run multiple instances of `grafana-db-exporter` with different `GRAFANA_SA_TOKEN` and different `GRAFANA_URL` in case of multiple instances. The token is the only way to authenticate with the Grafana API, and it is also the token that represents the organization that the service account belongs to, limiting the access to the dashboards to the ones that the organization has access to.

More info: [Grafana docs on Service accounts](https://grafana.com/docs/grafana/latest/administration/service-accounts/)

## Security Considerations

- Use read-only service accounts for Grafana access
- Rotate tokens regularly
- Use deploy keys with minimal repository access
- Consider network security between components

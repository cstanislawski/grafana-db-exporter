# grafana-db-exporter

Export your ClickOps'ed Grafana dashboards into the repository.

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
- `GRAFANA_SA_TOKEN`: API key / [Service Account token](https://grafana.com/docs/grafana/latest/administration/service-accounts/) (Viewer role is enough) to authenticate with the Grafana instance

Optional:

- `BASE_BRANCH`: Branch to create the PR against. Defaults to `main`,
- `BRANCH_PREFIX`: Prefix to use for the branch name. Defaults to `grafana-db-exporter-`
- `SSH_KEY_PASSWORD`: Passphrase to use to decrypt the SSH key. Defaults to `""`,
- `SSH_ACCEPT_UNKNOWN_HOSTS`: Whether to ignore unknown hosts when connecting to the Grafana instance, defaults to `false`

Conditional:

- `SSH_KNOWN_HOSTS_PATH`: The path to the known hosts file to use when connecting to the Grafana instance, required if `SSH_ACCEPT_UNKNOWN_HOSTS` is `false`,

### Examples

#### Docker

Set up the environment variables based on the configuration above as well as the `.env.example` file, and run the following command:

```bash
docker run --env-file .env ghcr.io/cstanislawski/grafana-db-exporter:latest
```

#### Kubernetes CronJob

```yaml
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
            - name: known-hosts
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
          - name: known-hosts
            secret:
              secretName: grafana-db-exporter
              items:
              - key: known_hosts
                path: known_hosts
          restartPolicy: OnFailure
```

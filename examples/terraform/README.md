# Local Development with Terraform

This example demonstrates how to use Terraform to manage Grafana dashboards, which is one of the possible use cases with `grafana-db-exporter`. The setup is split into two steps:

1. Infrastructure setup (local Grafana instance)
2. Dashboard management

## Overview

The `grafana-db-exporter` tool is designed to bridge the gap between UI-based dashboard creation and Infrastructure as Code (IaC). While this example shows how to manage dashboards with Terraform, the real power comes from combining both approaches:

1. **Dashboard Development**:
   - Create and iterate on dashboards using Grafana UI
   - Use `grafana-db-exporter` to automatically export changes
   - Commit changes to version control

2. **Dashboard Deployment**:
   - Use Terraform to apply dashboards across environments
   - Review changes in CI/CD pipelines
   - Maintain consistency across instances

## Local Development Setup

### Step 1: Start Local Grafana

First, set up your local Grafana instance:

```bash
cd infrastructure
terraform init
terraform apply
```

This will:

- Create a Docker network for Grafana
- Create a volume for persistence
- Start a Grafana container on port 3000

Verify Grafana is running:

```bash
# Check the URL and credentials
terraform output grafana_url
terraform output -json grafana_credentials

# Visit http://localhost:3000 and log in with admin/admin
```

### Step 2: Manage Dashboards

Once Grafana is running, you can manage dashboards from the root directory:

```bash
cd ..
terraform init
terraform apply
```

## Integration with CI/CD

The `grafana-db-exporter` tool is designed to be part of your CI/CD pipeline. Here's a typical workflow:

1. **Development Phase**:
   - Developers create/modify dashboards in Grafana UI
   - `grafana-db-exporter` runs periodically (via CronJob or CI) to capture changes
   - Changes are committed to a new branch
   - Pull/Merge Request is created automatically

2. **Review Phase**:
   - CI pipeline runs `terraform plan` showing dashboard changes
   - Reviewers can see exact modifications in JSON format
   - Changes can be tested in staging environments

3. **Deployment Phase**:
   - Upon merge, CD pipeline runs `terraform apply`
   - Dashboards are deployed to production Grafana
   - State is maintained in Terraform

Example CI/CD Pipeline:

```yaml
stages:
  - export
  - plan
  - apply

export_dashboards:
  stage: export
  script:
    - grafana-db-exporter  # Exports changes to git

terraform_plan:
  stage: plan
  script:
    - terraform init
    - terraform plan  # Shows dashboard changes

terraform_apply:
  stage: apply
  script:
    - terraform apply -auto-approve
  only:
    - main  # Only apply on main branch
```

## Best Practices

1. **Version Control**:
   - Always review dashboard changes in PRs
   - Use meaningful commit messages
   - Keep dashboard JSON files formatted consistently

2. **CI/CD Integration**:
   - Run `grafana-db-exporter` on a schedule
   - Include both export and apply steps
   - Maintain separate environments (dev/staging/prod)

3. **Dashboard Management**:
   - Use folders to organize dashboards
   - Maintain consistent naming conventions
   - Document dashboard purposes

## Directory Structure

```txt
.
├── README.md
├── dashboards/           # Dashboard JSON files
│   ├── adyon03rd3q4ge.json
│   └── some-folder/
│       └── ae1nxrr93p5ogd.json
├── infrastructure/       # Local Grafana setup
│   ├── main.tf
│   ├── outputs.tf
│   ├── terraform.tfvars
│   ├── variables.tf
│   └── versions.tf
├── main.tf              # Dashboard management
├── outputs.tf
├── terraform.tfvars
├── variables.tf
└── versions.tf
```

## Cleanup

To clean up:

```bash
# Remove dashboards first
terraform destroy

# Then remove the infrastructure
cd infrastructure
terraform destroy
```

## Next Steps

- Set up `grafana-db-exporter` in your CI/CD pipeline
- Configure automated PR creation for dashboard changes
- Implement environment-specific dashboard variations
- Add dashboard testing in CI pipeline

For more information on `grafana-db-exporter`, check the [main documentation](../README.md).

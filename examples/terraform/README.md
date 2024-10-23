# Terraform Example

This example demonstrates how to use Terraform to manage Grafana dashboards, which is one of the possible use cases with `grafana-db-exporter`. The setup is split into two steps:

1. Infrastructure setup - this will set up a local Grafana instance you can use for visualizing how Terraform would work with your production Grafana instance.
2. Dashboard management - this will create a few sample dashboards in the local Grafana instance.

## Overview

The `grafana-db-exporter` tool is designed to bridge the gap between UI-based dashboard creation and Infrastructure as Code (IaC). Grafana dashboards are inherently visual - they combine metrics, graphs and logs in a way that can be difficult to conceptualize through code alone. While this example shows how to manage dashboards with Terraform, the real power comes from combining both approaches:

1. **Dashboard Development**:
   - Create and iterate on dashboards using Grafana UI
   - See your changes in real-time as you adjust metrics, layouts, and visualizations
   - Fine-tune thresholds, colors, and alerts with immediate visual feedback
   - Use `grafana-db-exporter` to automatically export these visual changes to code
   - Commit changes to version control

2. **Infrastructure-as-Code Deployment**:
   - Use Terraform to apply these visually-perfected dashboards across environments
   - Review changes in CI/CD pipelines
   - Maintain consistency across instances

This hybrid approach gives you the best of both worlds: the immediate feedback and intuitive interface of Grafana's UI for design and iteration, combined with the reproducibility and automation of Infrastructure as Code for deployment.

## Setup

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

Once Grafana is running, you can create and manage dashboards through the UI. The visual editor allows you to:

- Drag and drop panels
- Adjust visualization settings
- Preview exactly how your dashboards will look

After you're satisfied with your changes, you can save them and they will be exported by the `grafana-db-exporter`.

## Integration with CI/CD

The `grafana-db-exporter` tool is designed to be part of your CI/CD pipeline, capturing your changes for automated deployment. Here's a typical workflow:

1. **Visual Development Phase**:
   - Developers create/modify dashboards in Grafana UI
   - See immediate feedback on changes
   - Iterate quickly on layouts and visualizations
   - `grafana-db-exporter` runs periodically (via CronJob or CI) to capture changes
   - Changes are committed to a new branch
   - Pull/Merge Request is created automatically

2. **Review Phase**:
   - CI pipeline runs `terraform plan` showing dashboard changes
   - Reviewers can see exact modifications in JSON format
   - Changes can be tested in staging environments where reviewers can see the actual dashboard

3. **Deployment Phase**:
   - Upon merge, CD pipeline runs `terraform apply`
   - Dashboards are deployed to production Grafana
   - State is maintained in Terraform
   - Visual consistency is guaranteed across environments

## Best Practices

1. **Dashboard Development**:
   - Use the Grafana UI for creation and redesigns
   - Test different visualization types to find the most effective way to present your data
   - Validate your changes in real-time
   - Use template variables to make dashboards more flexible

2. **Version Control**:
   - Always review dashboard changes in PRs
   - Use meaningful commit messages
   - Keep dashboard JSON files formatted consistently

3. **CI/CD Integration**:
   - Run `grafana-db-exporter` on a schedule
   - Include both export and apply steps
   - Maintain separate environments (dev/staging/prod)
   - Use test/staging environment to verify visual consistency

4. **Dashboard Management**:
   - Use folders to organize dashboards
   - Maintain consistent naming conventions
   - Document dashboard purposes
   - Keep related visualizations grouped together

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

- Set up `grafana-db-exporter` as part of your CI/CD pipeline to automate dashboard management - for example as a Kubernetes CronJob
- Configure automated PR creation for dashboard changes
- Implement environment-specific dashboard variations
- Add dashboard testing in CI pipeline

For more information on `grafana-db-exporter`, check the [main documentation](../README.md).

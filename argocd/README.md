# AWX Operator ArgoCD Deployment

This directory contains configuration files for deploying the AWX operator using ArgoCD.

## Single Application Deployment

For deploying to a single cluster, use the `awx-operator-application.yaml` file:

```bash
# Apply the Application resource to your ArgoCD instance
kubectl apply -f awx-operator-application.yaml -n argocd
```

This will create an ArgoCD Application that will deploy the AWX operator using kustomize.

## Multi-Environment Deployment with ApplicationSet

For deploying to multiple clusters or environments, use the `awx-operator-applicationset.yaml` file:

```bash
# Apply the ApplicationSet resource to your ArgoCD instance
kubectl apply -f awx-operator-applicationset.yaml -n argocd
```

This will create an ArgoCD ApplicationSet that will generate multiple Applications for different clusters and environments, based on the configurations in the file.

## Environment-Specific Configurations

The following environment-specific value files are provided:

- `values-production.yaml`: Configuration for production environments
- `values-dev.yaml`: Configuration for development environments

These files can be customized to match your specific requirements.

## Customizing the Deployment

To customize the deployment:

1. Modify the values in the appropriate values file
2. Update the repositories and paths in the Application or ApplicationSet file
3. Add or modify patches in the `patches` directory

## Manual Sync

You can manually sync the applications from the ArgoCD UI or using the ArgoCD CLI:

```bash
# Install the ArgoCD CLI
curl -sSL -o argocd https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-amd64
chmod +x argocd
sudo mv argocd /usr/local/bin/argocd

# Login to ArgoCD
argocd login <ARGOCD_SERVER>

# Sync the application
argocd app sync awx-operator
```

## Monitoring the Deployment

You can monitor the deployment status from the ArgoCD UI or using the ArgoCD CLI:

```bash
# Check the application status
argocd app get awx-operator

# Check the application history
argocd app history awx-operator
``` 
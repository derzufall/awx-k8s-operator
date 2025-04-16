# AWX Kubernetes Operator

This repository contains a Kubernetes operator for AWX (Ansible Web UI).

## Prerequisites

- Docker
- kubectl
- Helm (for manual deployments)
- Access to a Kubernetes cluster
- (Optional) kustomize (if not using kubectl's built-in kustomize)

## Building and Deploying

The `deploy.sh` script provides a convenient way to build, push, and deploy the operator.

### Environment Variables

You can customize the build and deployment by setting these environment variables:

- `REGISTRY`: Container registry (default: quay.io/wolkenzentrale)
- `IMAGE_NAME`: Image name (default: awx-operator)
- `TAG`: Image tag (default: aed406c)
- `NAMESPACE`: Namespace for deployment (default: awx-operator-system)

### Available Commands

```bash
# Build the operator image
./deploy.sh build

# Push the operator image to registry
./deploy.sh push

# Update Helm values file with image details
./deploy.sh update-values

# Install Custom Resource Definitions
./deploy.sh install-crd

# Deploy the operator to the Kubernetes cluster
./deploy.sh deploy

# Remove the operator from the Kubernetes cluster
./deploy.sh undeploy

# Run all commands in sequence (build, push, update-values, install-crd, deploy)
./deploy.sh all
```

### One-Command Deployment

The operator is designed to be deployed in a single command using Helm:

```bash
# Set your environment variables
export REGISTRY=your-registry.com
export IMAGE_NAME=awx-operator
export TAG=v1.0.0
export NAMESPACE=awx

# Update values and deploy
./deploy.sh update-values
./deploy.sh deploy
```

### Using ArgoCD

For GitOps deployments, we provide ArgoCD application configurations:

```bash
# Deploy with ArgoCD Application
kubectl apply -f argocd/awx-operator-application.yaml -n argocd

# Or deploy with ApplicationSet for multi-environment setups
kubectl apply -f argocd/awx-operator-applicationset.yaml -n argocd
```

### Customizing for Your Cluster

You can customize the deployment for your specific cluster:

1. Edit the values file:
   ```bash
   vi argocd/values.yaml
   ```

2. Apply the customized deployment:
   ```bash
   helm upgrade --install awx-operator ./argocd --namespace $NAMESPACE --create-namespace
   ```

## Creating an AWX Instance

After the operator is deployed, you can create an AWX instance by creating a custom resource:

```yaml
# example-awx.yaml
apiVersion: awx.ansible.com/v1beta1
kind: AWXInstance
metadata:
  name: example-awx
spec:
  version: latest
```

Apply it with:

```bash
kubectl apply -f example-awx.yaml
```

## State Reconciliation

The operator maintains the desired state of AWX resources through two mechanisms:

1. **Resource Watch**: The operator watches for changes to AWXInstance resources and ensures that the AWX state always matches the configured state.

2. **Internal State Check**: Every 60 seconds (configurable via `operator.reconciliation.period` in values.yaml), the operator checks if the state was changed internally within AWX. If any deviation is detected, the operator automatically corrects the changes to match the desired state specified in the AWXInstance resource.

This ensures that even if changes are made directly in the AWX UI or API, the operator will detect and revert those changes to maintain the desired configuration.

## Development

To make changes to the operator:

1. Modify the controller code
2. Build and push a new image
3. Update the deployment with the new image
4. Deploy the updated operator

## Troubleshooting

If you encounter issues:

- Check the operator logs: `kubectl logs -l control-plane=controller-manager -n awx-operator-system`
- Verify CRDs are installed: `kubectl get crds | grep awx`
- Ensure RBAC permissions are correct: `kubectl get clusterrole,clusterrolebinding -l control-plane=controller-manager` 
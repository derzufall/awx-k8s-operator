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

## Managing an Existing AWX Instance

You can use this operator to manage an existing AWX instance that was deployed using the official AWX operator or by other means:

```yaml
# existing-awx.yaml
apiVersion: awx.ansible.com/v1alpha1
kind: AWXInstance
metadata:
  name: existing-awx
spec:
  hostname: awx.example.com
  protocol: https  # Optional: can be 'http' or 'https' (defaults to https)
  adminUser: admin
  adminPassword: yourpassword
  adminEmail: admin@example.com
  externalInstance: true  # This indicates we're connecting to an existing instance
  
  # Define resources to manage (projects, inventories, job templates)
  projects:
    - name: My Project
      scmType: git
      scmUrl: https://github.com/example/ansible-playbooks.git
```

For HTTP connections (if your AWX instance doesn't use HTTPS):

```yaml
# http-awx-instance.yaml
apiVersion: awx.ansible.com/v1alpha1
kind: AWXInstance
metadata:
  name: my-existing-awx
  namespace: awx-operator-system
spec:
  hostname: awx.example.com  # Your AWX host
  protocol: http  # Use HTTP if your AWX uses HTTP
  adminUser: admin
  adminPassword: yourpassword
  adminEmail: admin@example.com
  externalInstance: true  # Important - indicates this is an existing instance
```

Apply it with:

```bash
kubectl apply -f your-awx-instance.yaml
```
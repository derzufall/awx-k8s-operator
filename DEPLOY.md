# AWX Kubernetes Operator Deployment Guide

This guide explains how to deploy the AWX Kubernetes Operator.

## Prerequisites

- Docker (for building the operator image)
- Kubernetes cluster (v1.19+)
- kubectl (configured to access your cluster)
- Container registry access (like quay.io, Docker Hub)

## Configuration

Before deployment, configure the following environment variables:

- `REGISTRY`: Container registry (default: quay.io/myuser)
- `IMAGE_NAME`: Image name (default: awx-operator)
- `TAG`: Image tag (default: latest)
- `NAMESPACE`: Kubernetes namespace for deployment (default: awx-operator-system)

Example:
```bash
export REGISTRY=quay.io/myorg
export IMAGE_NAME=awx-operator
export TAG=0.1.0
export NAMESPACE=my-awx-operator
```

## Deployment Steps

### 0. All-in-one Deployment

To execute all deployment steps at once:

```bash
./deploy.sh all
```

This will build the image, push it to the registry, update kustomization files, install CRDs, and deploy the operator.

### 1. Build the Operator Image

```bash
./deploy.sh build
```

### 2. Push to Container Registry

```bash
./deploy.sh push
```

### 3. Update Kustomization Configuration

Updates the kustomization files with your image details:

```bash
./deploy.sh update-kustomization
```

### 4. Install Custom Resource Definitions

```bash
./deploy.sh install-crd
```

### 5. Deploy the Operator

```bash
./deploy.sh deploy
```

## Verification

Verify the deployment:

```bash
kubectl get pods -n $NAMESPACE
kubectl get awx -n $NAMESPACE
```

## Usage

After deployment, create an AWX instance by applying a custom resource:

```bash
kubectl apply -f config/samples/awx_v1beta1_awx.yaml -n $NAMESPACE
```

## Troubleshooting

If you encounter issues:

- Check operator logs: `kubectl logs -f deployment/awx-operator-controller-manager -n $NAMESPACE`
- Verify CRDs are installed: `kubectl get crds | grep awx`
- Check for events: `kubectl get events -n $NAMESPACE`

## Uninstalling

To remove the operator:

```bash
./deploy.sh undeploy
```

## Advanced Configuration

For advanced configurations, refer to the `config/samples` directory for example custom resources.

## Next Steps

- Configure persistent storage for AWX
- Set up ingress for external access
- Configure authentication 
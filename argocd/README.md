# AWX Operator ArgoCD Deployment

This directory contains the Helm chart and ArgoCD configuration for deploying the AWX Kubernetes Operator.

## Getting Started

### Prerequisites

- Kubernetes cluster
- ArgoCD installed on your cluster
- kubectl configured to access your cluster

### Installing

1. Apply the ArgoCD Application:

```bash
kubectl apply -f awx-operator-application.yaml -n argocd
```

This will create an ArgoCD Application that will deploy the AWX operator using Helm.

### Multi-Environment Deployment

For deploying to multiple environments, you can use the ApplicationSet:

```bash
kubectl apply -f awx-operator-applicationset.yaml -n argocd
```

This will create applications for each environment defined in the ApplicationSet.

## Configuration

You can customize the deployment by modifying the `values.yaml` file or by creating environment-specific value files such as `values-production.yaml` or `values-dev.yaml`.

The main configurable parameters are:

- `operator.image`: Container image configuration
- `operator.resources`: CPU and memory resource limits
- `operator.reconciliation.period`: Reconciliation interval in seconds
- `namespace`: Target namespace for the operator

## Using the Operator

After the operator is deployed, you can create AWX instances by applying AWXInstance custom resources:

```yaml
apiVersion: awx.ansible.com/v1alpha1
kind: AWXInstance
metadata:
  name: my-awx
  namespace: awx
spec:
  adminUser: admin
  adminPassword: password123
  adminEmail: admin@example.com
  hostname: awx.example.com
```

Apply this resource to your cluster:

```bash
kubectl apply -f your-awx-instance.yaml
```

## Troubleshooting

To check the status of your ArgoCD application:

```bash
kubectl describe application awx-operator -n argocd
```

To check the operator logs:

```bash
kubectl logs -l app=awx-operator -n <namespace>
``` 
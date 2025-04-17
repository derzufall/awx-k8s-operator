# AWX Kubernetes Operator - Java Implementation

This is a Java Spring Boot alternative implementation of the AWX Kubernetes Operator. It provides equivalent functionality to the Go version.

## Prerequisites

- JDK 17 or later
- Maven 3.8+
- Kubernetes cluster (for deployment)
- AWX instance (if using external mode)

## Building

To build the operator:

```bash
cd javaalternative
mvn clean package
```

## Running Locally

For local development, ensure you have a kubeconfig file in your `~/.kube/config` path:

```bash
java -jar target/awx-operator-0.1.0.jar
```

## Building with Docker

To build a Docker image:

```bash
docker build -t awx-operator-java:latest .
```

## Deploying to Kubernetes

1. Build the Docker image and push it to a registry
2. Apply the Kubernetes manifests:

```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/rbac.yaml
kubectl apply -f k8s/crd.yaml
kubectl apply -f k8s/deployment.yaml
```

## Creating AWX Resources

Apply an AWXInstance custom resource:

```yaml
apiVersion: awx.ansible.com/v1alpha1
kind: AWXInstance
metadata:
  name: my-awx
  namespace: awx-operator-system
spec:
  hostname: "awx.example.com"
  adminUser: "admin"
  adminPassword: "password"
  adminEmail: "admin@example.com"
  protocol: "https"
  projects:
    - name: "My Project"
      description: "My Project Description"
      scmType: "git"
      scmUrl: "https://github.com/example/repository.git"
      scmBranch: "main"
  inventories:
    - name: "My Inventory"
      description: "My Inventory Description"
      hosts:
        - name: "host1.example.com"
          description: "Host 1"
  jobTemplates:
    - name: "My Job Template"
      description: "My Job Template Description"
      projectName: "My Project"
      inventoryName: "My Inventory"
      playbook: "playbook.yml"
```

## Features

- Manages AWX Projects, Inventories, and Job Templates
- Handles resource finalizers for clean deletion
- Reconnection handling
- Status condition tracking

## Architecture

The operator is built with:

- Spring Boot framework
- Official Kubernetes Java client
- Spring DI for dependency management
- Controller-based reconciliation pattern

## Status

This is an alternative implementation of the Go-based AWX Kubernetes Operator and is provided as a reference. 
#!/bin/bash

# Default values
REGISTRY=${REGISTRY:-quay.io/wolkenzentrale}
IMAGE_NAME=${IMAGE_NAME:-awx-operator}
TAG=${TAG:-aed406c}
NAMESPACE=${NAMESPACE:-awx-operator-system}

# Colors for better output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print header
print_header() {
  echo -e "\n${BLUE}==== $1 ====${NC}\n"
}

# Function to print info
print_info() {
  echo -e "${GREEN}INFO:${NC} $1"
}

# Function to print error
print_error() {
  echo -e "${RED}ERROR:${NC} $1"
}

# Check prerequisites
check_prerequisites() {
  print_header "Checking prerequisites"

  # Check for Docker
  if ! command -v docker &> /dev/null; then
    print_error "Docker not found. Please install Docker first."
    exit 1
  fi

  # Check for kubectl
  if ! command -v kubectl &> /dev/null; then
    print_error "kubectl not found. Please install kubectl first."
    exit 1
  fi

  # Check for helm
  if ! command -v helm &> /dev/null; then
    print_error "helm not found. Please install Helm first."
    exit 1
  fi

  print_info "All prerequisites satisfied"
}

# Build the operator image
build() {
  print_header "Building operator image"
  docker build -t "${REGISTRY}/${IMAGE_NAME}:${TAG}" .
  echo "Image built successfully: ${REGISTRY}/${IMAGE_NAME}:${TAG}"
}

# Push the operator image to the registry
push() {
  print_header "Pushing operator image"
  docker push "${REGISTRY}/${IMAGE_NAME}:${TAG}"
  echo "Image pushed successfully: ${REGISTRY}/${IMAGE_NAME}:${TAG}"
}

# Update Helm values file with image details
update_values() {
  print_header "Updating Helm values"
  
  cat > argocd/values.yaml << EOF
operator:
  image:
    registry: ${REGISTRY}
    repository: ${IMAGE_NAME}
    tag: ${TAG}
    pullPolicy: IfNotPresent
  
  resources:
    limits:
      cpu: 500m
      memory: 512Mi
    requests:
      cpu: 200m
      memory: 256Mi
  
  reconciliation:
    period: 60  # in seconds
  
  logs:
    level: info

# Namespace settings
namespace: ${NAMESPACE}
createNamespace: true

# ArgoCD application settings
argocd:
  project: default
  server: https://kubernetes.default.svc
  automated:
    prune: true
    selfHeal: true
EOF

  echo "Updated Helm values file"
}

install_crd() {
  print_header "Installing Custom Resource Definitions"
  kubectl apply -f argocd/templates/crd.yaml
  echo "CRDs installed"
}

deploy() {
  print_header "Deploying operator"
  
  # Create namespace if it doesn't exist
  kubectl get namespace "${NAMESPACE}" &> /dev/null || kubectl create namespace "${NAMESPACE}"
  
  # Apply using Helm
  helm upgrade --install awx-operator ./argocd \
    --namespace "${NAMESPACE}" \
    --create-namespace \
    --values argocd/values.yaml
  
  echo "Operator deployed to namespace: ${NAMESPACE}"
  echo "Waiting for operator pod to be ready..."
  
  # Wait for the pod to be ready
  kubectl wait --for=condition=ready pod -l app=awx-operator -n "${NAMESPACE}" --timeout=120s
  
  echo "Operator is ready!"
}

undeploy() {
  print_header "Undeploying operator"
  
  # Uninstall Helm release
  helm uninstall awx-operator -n "${NAMESPACE}" || true
  
  # Delete namespace
  kubectl delete namespace "${NAMESPACE}" || true
  
  echo "Operator undeployed"
}

show_help() {
  echo "Usage: $0 [command]"
  echo ""
  echo "Commands:"
  echo "  build                Build the operator image"
  echo "  push                 Push the operator image to registry"
  echo "  update-values        Update Helm values file with image details"
  echo "  install-crd          Install Custom Resource Definitions"
  echo "  deploy               Deploy the operator to the Kubernetes cluster"
  echo "  undeploy             Remove the operator from the Kubernetes cluster"
  echo "  all                  Run all commands in sequence (build, push, update-values, install-crd, deploy)"
  echo ""
  echo "Environment variables:"
  echo "  REGISTRY             Container registry (default: quay.io/wolkenzentrale)"
  echo "  IMAGE_NAME           Image name (default: awx-operator)"
  echo "  TAG                  Image tag (default: aed406c)"
  echo "  NAMESPACE            Namespace for deployment (default: awx-operator-system)"
}

# Main execution
case "$1" in
  build)
    check_prerequisites
    build
    ;;
  push)
    check_prerequisites
    push
    ;;
  update-values)
    check_prerequisites
    update_values
    ;;
  install-crd)
    check_prerequisites
    install_crd
    ;;
  deploy)
    check_prerequisites
    deploy
    ;;
  undeploy)
    check_prerequisites
    undeploy
    ;;
  all)
    check_prerequisites
    build
    push
    update_values
    install_crd
    deploy
    ;;
  *)
    show_help
    ;;
esac

# Make script executable
# chmod +x deploy.sh 
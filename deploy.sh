#!/bin/bash
set -e

# Default values
REGISTRY=${REGISTRY:-"quay.io/myuser"}
IMAGE_NAME=${IMAGE_NAME:-"awx-operator"}
TAG=${TAG:-"latest"}
IMAGE="${REGISTRY}/${IMAGE_NAME}:${TAG}"
NAMESPACE=${NAMESPACE:-"awx-operator-system"}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Helper functions
print_header() {
  echo -e "\n${GREEN}==== $1 ====${NC}\n"
}

print_error() {
  echo -e "${RED}ERROR: $1${NC}"
}

print_info() {
  echo -e "${YELLOW}INFO: $1${NC}"
}

check_prerequisites() {
  print_header "Checking prerequisites"
  
  # Check for Docker
  if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed"
    exit 1
  fi
  
  # Check for kubectl
  if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed"
    exit 1
  fi
  
  # Check for kustomize
  if ! command -v kustomize &> /dev/null; then
    print_info "kustomize not found, will use kubectl's built-in kustomize"
  fi

  echo "Docker: $(docker --version)"
  echo "kubectl: $(kubectl version --client --short)"
  
  print_info "Using image: ${IMAGE}"
}

build() {
  print_header "Building operator image"
  docker build -t "${IMAGE}" .
  echo "Image built: ${IMAGE}"
}

push() {
  print_header "Pushing image to registry"
  docker push "${IMAGE}"
  echo "Image pushed: ${IMAGE}"
}

update_kustomization() {
  print_header "Updating kustomization files"
  
  # Create root kustomization.yaml if it doesn't exist
  if [ ! -f "kustomization.yaml" ]; then
    cat > kustomization.yaml << EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

# Points to the default kustomization which includes all needed components
resources:
- config/default

# Namespace configuration
namespace: ${NAMESPACE}

# Image configuration - these will override the defaults in config/manager/kustomization.yaml
images:
- name: controller
  newName: ${REGISTRY}/${IMAGE_NAME}
  newTag: ${TAG}

# Common labels for all resources
commonLabels:
  app.kubernetes.io/name: awx-operator
  app.kubernetes.io/instance: awx-operator
  app.kubernetes.io/part-of: awx-operator
  app.kubernetes.io/managed-by: kustomize
EOF
  else
    # Update only the image configuration in existing kustomization.yaml
    # This is a bit complex with sed, so we'll use a temporary file
    tmp_file=$(mktemp)
    cat kustomization.yaml | 
      sed -E "s|newName: .*|newName: ${REGISTRY}/${IMAGE_NAME}|g" | 
      sed -E "s|newTag: .*|newTag: ${TAG}|g" |
      sed -E "s|namespace: .*|namespace: ${NAMESPACE}|g" > $tmp_file
    mv $tmp_file kustomization.yaml
  fi
  
  # Create manager resources patch directory if it doesn't exist
  mkdir -p patches

  # Create or update manager resources patch
  cat > patches/manager_resources.yaml << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: awx-operator-controller-manager
  namespace: system
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: manager
        resources:
          limits:
            cpu: 500m
            memory: 500Mi
          requests:
            cpu: 200m
            memory: 200Mi
EOF

  echo "Updated kustomization.yaml and patches"
}

install_crd() {
  print_header "Installing Custom Resource Definitions"
  kubectl apply -f config/crd/bases/
  echo "CRDs installed"
}

deploy() {
  print_header "Deploying operator"
  
  # Create namespace if it doesn't exist
  kubectl get namespace "${NAMESPACE}" &> /dev/null || kubectl create namespace "${NAMESPACE}"
  
  # Apply using kustomize
  kubectl apply -k .
  
  echo "Operator deployed to namespace: ${NAMESPACE}"
  echo "Waiting for operator pod to be ready..."
  
  # Wait for the pod to be ready
  kubectl wait --for=condition=ready pod -l control-plane=controller-manager -n "${NAMESPACE}" --timeout=120s
  
  echo "Operator is ready!"
}

undeploy() {
  print_header "Undeploying operator"
  
  # Delete operator resources using kustomize
  kubectl delete -k . || true
  
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
  echo "  update-kustomization Update kustomization files with image details"
  echo "  install-crd          Install Custom Resource Definitions"
  echo "  deploy               Deploy the operator to the Kubernetes cluster"
  echo "  undeploy             Remove the operator from the Kubernetes cluster"
  echo "  all                  Run all commands in sequence (build, push, update-kustomization, install-crd, deploy)"
  echo ""
  echo "Environment variables:"
  echo "  REGISTRY             Container registry (default: quay.io/myuser)"
  echo "  IMAGE_NAME           Image name (default: awx-operator)"
  echo "  TAG                  Image tag (default: latest)"
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
  update-kustomization)
    check_prerequisites
    update_kustomization
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
    update_kustomization
    install_crd
    deploy
    ;;
  *)
    show_help
    ;;
esac

# Make script executable
# chmod +x deploy.sh 
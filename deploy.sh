#!/bin/bash
#
# Deploys the Nephoran Intent Operator to a Kubernetes cluster.
#
# This script supports two deployment environments: 'local' and 'remote'.
#
# USAGE:
#   ./deploy.sh [local|remote]
#
# ARGUMENTS:
#   local   - Builds images and deploys them to the current Kubernetes context
#             using the 'Never' imagePullPolicy. Ideal for local development
#             with Minikube or Docker Desktop.
#   remote  - Builds and pushes images to Google Artifact Registry, then deploys
#             to the current Kubernetes context. It patches the service account
#             to use an imagePullSecret.
#
# REQUIREMENTS:
#   - docker, kubectl, kustomize, gcloud CLI tools
#   - For 'remote': Authenticated to GCP and configured Docker credentials.
#

set -eo pipefail

# --- Configuration ---
# Replace with your GCP Project ID and Artifact Registry location.
GCP_PROJECT_ID="poised-elf-466913-q2"
GCP_REGION="us-central1"
AR_REPO="nephoran"

# --- Script Logic ---
ENV=$1
if [[ -z "$ENV" ]]; then
  echo "Error: Deployment environment not specified."
  echo "Usage: $0 [local|remote]"
  exit 1
fi

# Image definitions
declare -A IMAGES
IMAGES=(
  ["llm-processor"]="cmd/llm-processor"
  ["nephio-bridge"]="cmd/nephio-bridge"
  ["oran-adaptor"]="cmd/oran-adaptor"
  ["rag-api"]="pkg/rag"
)
IMAGE_TAG=$(git rev-parse --short HEAD)
KUSTOMIZE_DIR="deployments/kustomize/overlays/${ENV}"

# --- Functions ---
build_images() {
  echo "Building container images with tag: ${IMAGE_TAG}..."
  for image_name in "${!IMAGES[@]}"; do
    docker_path="${IMAGES[$image_name]}"
    full_image_name="${image_name}:${IMAGE_TAG}"
    echo "Building ${full_image_name} from ${docker_path}..."
    docker build -t "${full_image_name}" -f "${docker_path}/Dockerfile" .
  done
}

load_images_into_local_cluster() {
  echo "Loading images into local Kubernetes cluster..."
  # Detect local cluster type from kubectl context
  local context
  context=$(kubectl config current-context)

  for image_name in "${!IMAGES[@]}"; do
    full_image_name="${image_name}:${IMAGE_TAG}"
    echo "Loading ${full_image_name}..."
    if [[ "$context" == "minikube" ]]; then
      minikube image load "${full_image_name}"
    elif [[ "$context" == "kind-"* ]]; then
      # kind context is often prefixed with 'kind-'
      kind load docker-image "${full_image_name}"
    else
      echo "Warning: Unsupported or unknown local K8s context '${context}'."
      echo "Your Docker environment may be shared with Kubernetes, so this may not be an issue."
      echo "If you see 'ErrImageNeverPull', please load the image manually:"
      echo "  - For Minikube: minikube image load ${full_image_name}"
      echo "  - For Kind:     kind load docker-image ${full_image_name}"
    fi
  done
}

push_images() {
  echo "Pushing images to Artifact Registry..."
  AR_URL="${GCP_REGION}-docker.pkg.dev/${GCP_PROJECT_ID}/${AR_REPO}"
  for image_name in "${!IMAGES[@]}"; do
    local_image="${image_name}:${IMAGE_TAG}"
    remote_image="${AR_URL}/${image_name}:${IMAGE_TAG}"
    echo "Tagging and pushing ${remote_image}"
    docker tag "${local_image}" "${remote_image}"
    docker push "${remote_image}"

    # Update kustomization with the new image tag
    # The image name in the kustomization.yaml is the full remote path.
    image_in_kustomization="${AR_URL}/${image_name}"
    echo "Updating kustomization for ${image_name}..."
    (cd "${KUSTOMIZE_DIR}" && kustomize edit set image "${image_in_kustomization}=${remote_image}")
  done
}

deploy_kubernetes() {
  echo "Deploying to Kubernetes using kustomize overlay: ${KUSTOMIZE_DIR}"
  kubectl apply -k "${KUSTOMIZE_DIR}"

  echo "Restarting deployments to apply changes..."
  for image_name in "${!IMAGES[@]}"; do
    kubectl rollout restart deployment "${image_name}"
  done
  echo "Deployment complete."
}

# --- Main Execution ---
echo "Starting deployment for environment: ${ENV}"

build_images

if [[ "$ENV" == "remote" ]]; then
  # Authenticate Docker to Artifact Registry
  gcloud auth configure-docker "${GCP_REGION}-docker.pkg.dev"
  push_images
elif [[ "$ENV" == "local" ]]; then
  load_images_into_local_cluster
  echo "Skipping image push for local deployment."
  # For local, we just need to set the tag. The name is already set in the overlay.
  for image_name in "${!IMAGES[@]}"; do
    (cd "${KUSTOMIZE_DIR}" && kustomize edit set image "${image_name}=${image_name}:${IMAGE_TAG}")
  done
else
  echo "Error: Invalid environment '${ENV}'. Use 'local' or 'remote'."
  exit 1
fi

deploy_kubernetes
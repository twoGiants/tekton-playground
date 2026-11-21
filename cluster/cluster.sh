#!/usr/bin/env bash
set -e -o pipefail

declare TEKTON_PIPELINE_VERSION TEKTON_TRIGGERS_VERSION TEKTON_DASHBOARD_VERSION
declare HELP CLUSTER_CONFIG SKIP_TEKTON_INSTALL LOCAL_PIPELINE_SRC K8_NODE_IMAGE

get_latest_release() {
  # Follow the redirect of the releases/latest URL to extract the tag (avoids GitHub API rate limits)
  curl --silent --output /dev/null --write-out '%{redirect_url}' "https://github.com/$1/releases/latest" |
    sed -E 's|.*/tag/||'
}

get_latest_kind_image() {
  # The rendered release notes of the latest kind release contain the node image
  # incl. digest; the default node image is listed first (avoids GitHub API rate limits)
  curl --silent --location "https://github.com/kubernetes-sigs/kind/releases/latest" |
    grep -oE 'kindest/node:v[0-9]+\.[0-9]+\.[0-9]+@sha256:[a-f0-9]{64}' |
    head -n 1
}

info() {
  echo -e "[\e[93mINFO\e[0m] $1"
}

error() {
  echo -e "[\e[91mERROR\e[0m] $1"
}


default_cluster_config() {
  cat <<'EOF'
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    image: "${image}"
  - role: worker
    image: "${image}"
  - role: worker
    image: "${image}"
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${reg_port}"]
    endpoint = ["http://${reg_name}:${reg_port}"]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."${reg_name}:${reg_port}"]
    endpoint = ["http://${reg_name}:${reg_port}"]
EOF
}

check_defaults() {
  info "Check and defaults input params..."
  export KIND_CLUSTER_NAME=${CLUSTER_NAME:-"tekton"}

  if [ -z "$TEKTON_PIPELINE_VERSION" ]; then
    TEKTON_PIPELINE_VERSION=$(get_latest_release tektoncd/pipeline)
  fi
  if [ -z "$TEKTON_TRIGGERS_VERSION" ]; then
    TEKTON_TRIGGERS_VERSION=$(get_latest_release tektoncd/triggers)
  fi
  if [ -z "$TEKTON_DASHBOARD_VERSION" ]; then
    TEKTON_DASHBOARD_VERSION=$(get_latest_release tektoncd/dashboard)
  fi
  if [ -z "$CONTAINER_RUNTIME" ]; then
    CONTAINER_RUNTIME="docker"
  fi
  if [ -z "$K8_NODE_IMAGE" ]; then
    K8_NODE_IMAGE=$(get_latest_kind_image)
  fi

  info "Using container runtime: $CONTAINER_RUNTIME"

  # Support environment variable fallback
  export LOCAL_PIPELINE_SRC=${LOCAL_PIPELINE_SRC:-$TEKTON_PIPELINE_SRC}

  # Check for local pipeline source
  if [ -n "$LOCAL_PIPELINE_SRC" ]; then
    if ! command -v ko &> /dev/null; then
      error "'ko' command not found. Install from https://ko.build"
      exit 1
    fi

    if [ ! -d "$LOCAL_PIPELINE_SRC" ]; then
      error "Local pipeline source directory not found: $LOCAL_PIPELINE_SRC"
      exit 1
    fi

    if [ ! -d "$LOCAL_PIPELINE_SRC/config" ]; then
      error "config/ directory not found in: $LOCAL_PIPELINE_SRC"
      exit 1
    fi

    info "Using local Tekton Pipeline source: $LOCAL_PIPELINE_SRC"
  fi
}

create_registry() {
  info "Checking if registry exists..."
  reg_name='kind-registry'
  reg_port='5000'
  running="$(${CONTAINER_RUNTIME} inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
  if [ "${running}" == 'true' ]; then
    info "Registry exists..."
    return 0
  fi

  info "Registry does not exist, creating..."
  "$CONTAINER_RUNTIME" rm "${reg_name}" 2>/dev/null || true
  "$CONTAINER_RUNTIME" run \
    -d \
    --restart=always \
    -p "${reg_port}:5000" \
    --name "${reg_name}" \
    registry:2
  info "Registry started..."
}

load_cluster_config() {
  local config
  if [ -n "$CLUSTER_CONFIG" ]; then
    config=$(sed "s/\${reg_name}/${reg_name}/g; s/\${reg_port}/${reg_port}/g; s|\${image}|${K8_NODE_IMAGE}|g" "$CLUSTER_CONFIG")
    echo "$config"
    return 0
  fi

  config=$(default_cluster_config | sed "s/\${reg_name}/${reg_name}/g; s/\${reg_port}/${reg_port}/g; s|\${image}|${K8_NODE_IMAGE}|g")
  echo "$config"
}

create_cluster() {
  info "Checking if cluster exists..."
  running_cluster=$(kind get clusters | grep "$KIND_CLUSTER_NAME" || true)
  if [ "${running_cluster}" == "$KIND_CLUSTER_NAME" ]; then
    info "Cluster exists..."
    return 0
  fi

  info "Cluster does not exist, creating with the local registry enabled in containerd..."
  kind create cluster --config=<(load_cluster_config)
  info "Waiting for the nodes to be ready..."
  kubectl wait --for=condition=ready node --all --timeout=600s
}

connect_registry() {
  info "Check if registry is connected to the cluster network..."
  connected_registry=$("$CONTAINER_RUNTIME" network inspect kind -f '{{json .Containers}}' | grep -q "${reg_name}" && echo "true" || echo "false")
  if [ "${connected_registry}" == 'true' ]; then
    info "Registry is connected..."
    return 0
  fi

  info "Registry is not connected, connecting the registry to the cluster network..."
  "$CONTAINER_RUNTIME" network connect "kind" "${reg_name}" || true
  info "Connection established..."
}

install_tekton() {
  info "Checking if Tekton is installed in the cluster..."
  running_tekton=$(kubectl get crds | grep -q "pipelines.tekton.dev" && echo "true" || echo "false")
  if [ "${running_tekton}" == 'true' ]; then
    info "Tekton is installed..."
    info "Tekton Dashboard available at http://localhost:9097"
    return 0
  fi

  info "Tekton is not installed, installing Tekton Pipeline, Triggers and Dashboard..."

  if [ -n "$LOCAL_PIPELINE_SRC" ]; then
    info "Building and deploying Tekton Pipeline from local source: ${LOCAL_PIPELINE_SRC}..."
    (cd "${LOCAL_PIPELINE_SRC}" && ko apply -R -f config/)
  else
    info "Installing Tekton Pipeline from remote release..."
    kubectl apply -f https://github.com/tektoncd/pipeline/releases/download/"${TEKTON_PIPELINE_VERSION}"/release.yaml
  fi

  kubectl apply -f https://github.com/tektoncd/triggers/releases/download/"${TEKTON_TRIGGERS_VERSION}"/release.yaml
  kubectl wait --for=condition=Established --timeout=30s crds/clusterinterceptors.triggers.tekton.dev || true # Starting from triggers v0.13
  kubectl apply -f https://github.com/tektoncd/triggers/releases/download/"${TEKTON_TRIGGERS_VERSION}"/interceptors.yaml || true
  kubectl apply -f https://github.com/tektoncd/dashboard/releases/download/"${TEKTON_DASHBOARD_VERSION}"/release-full.yaml

  info "Wait until all pods are ready..."
  kubectl wait -n tekton-pipelines --for=condition=ready pods --all --timeout=600s
  kubectl port-forward service/tekton-dashboard -n tekton-pipelines 9097:9097 &>/tmp/kind-tekton-dashboard.log &
  info "Tekton Dashboard available at http://localhost:9097"
}

help_msg() {
  cat <<'EOF'
Usage: cluster.sh [options]

Options:
  -h                    Show this help and exit
  -c <name>             Cluster name (default: tekton)
  -p <version>          Tekton Pipeline version (default: latest)
  -t <version>          Tekton Triggers version (default: latest)
  -d <version>          Tekton Dashboard version (default: latest)
  -l <path>             Local Tekton Pipeline source, built and deployed with ko
  -s                    Skip Tekton installation

Environment variables (options take precedence):
  CLUSTER_NAME             kind cluster name (default: tekton)
  K8_NODE_IMAGE            kind node image incl. @sha256 digest
                           (default: latest from kind releases)
  CLUSTER_CONFIG           path to custom kind cluster config; the placeholders
                           ${reg_name}, ${reg_port} and ${image} are substituted
  CONTAINER_RUNTIME        container runtime: docker or podman (default: docker)
  TEKTON_PIPELINE_VERSION  Tekton Pipeline version (default: latest)
  TEKTON_TRIGGERS_VERSION  Tekton Triggers version (default: latest)
  TEKTON_DASHBOARD_VERSION Tekton Dashboard version (default: latest)
  LOCAL_PIPELINE_SRC       local Tekton Pipeline source path
                           (fallback: TEKTON_PIPELINE_SRC)
EOF
}

while getopts ":hc:p:t:d:l:s" opt; do
  case ${opt} in
  h)
    help_msg
    exit 0
    ;;
  c)
    CLUSTER_NAME=$OPTARG
    ;;
  p)
    TEKTON_PIPELINE_VERSION=$OPTARG
    ;;
  t)
    TEKTON_TRIGGERS_VERSION=$OPTARG
    ;;
  d)
    TEKTON_DASHBOARD_VERSION=$OPTARG
    ;;
  l)
    LOCAL_PIPELINE_SRC=$OPTARG
    ;;
  s)
    SKIP_TEKTON_INSTALL=true
    ;;
  \?)
    echo "Invalid option: $OPTARG" 1>&2
    echo 1>&2
    help_msg
    exit 1
    ;;
  :)
    echo "Invalid option: $OPTARG requires an argument" 1>&2
    exit 1
    ;;
  esac
done
shift $((OPTIND - 1))

check_defaults
create_registry
create_cluster
connect_registry

if [ -z "$SKIP_TEKTON_INSTALL" ]; then
  install_tekton
else
  info "Skipping Tekton installation..."
fi

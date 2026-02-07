#!/usr/bin/env bash
set -e -o pipefail

declare TEKTON_PIPELINE_VERSION TEKTON_TRIGGERS_VERSION TEKTON_DASHBOARD_VERSION CLUSTER_CONFIG SKIP_TEKTON_INSTALL LOCAL_PIPELINE_SRC

get_latest_release() {
  curl --silent "https://api.github.com/repos/$1/releases/latest" |
    grep '"tag_name":' |
    sed -E 's/.*"([^"]+)".*/\1/'
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
  - role: worker
  - role: worker
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
    config=$(sed "s/\${reg_name}/${reg_name}/g; s/\${reg_port}/${reg_port}/g" "$CLUSTER_CONFIG")
    echo "$config"
    return 0
  fi

  config=$(default_cluster_config | sed "s/\${reg_name}/${reg_name}/g; s/\${reg_port}/${reg_port}/g")
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
    kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/previous/"${TEKTON_PIPELINE_VERSION}"/release.yaml
  fi

  kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/previous/"${TEKTON_TRIGGERS_VERSION}"/release.yaml
  kubectl wait --for=condition=Established --timeout=30s crds/clusterinterceptors.triggers.tekton.dev || true # Starting from triggers v0.13
  kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/previous/"${TEKTON_TRIGGERS_VERSION}"/interceptors.yaml || true
  kubectl apply -f https://github.com/tektoncd/dashboard/releases/download/"${TEKTON_DASHBOARD_VERSION}"/release-full.yaml

  info "Wait until all pods are ready..."
  kubectl wait -n tekton-pipelines --for=condition=ready pods --all --timeout=600s
  kubectl port-forward service/tekton-dashboard -n tekton-pipelines 9097:9097 &>/tmp/kind-tekton-dashboard.log &
  info "Tekton Dashboard available at http://localhost:9097"
}

while getopts ":c:p:t:d:l:s" opt; do
  case ${opt} in
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
    echo "Usage: cluster.sh [-c cluster-name -p pipeline-version -t triggers-version -d dashboard-version -l local-pipeline-path -s]"
    ;;
  :)
    echo "Invalid option: $OPTARG requires an argument" 1>&2
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

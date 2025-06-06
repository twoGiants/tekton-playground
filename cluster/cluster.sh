#!/usr/bin/env bash
set -e -o pipefail

declare TEKTON_PIPELINE_VERSION TEKTON_TRIGGERS_VERSION TEKTON_DASHBOARD_VERSION CLUSTER_CONFIG SKIP_TEKTON_INSTALL

get_latest_release() {
  curl --silent "https://api.github.com/repos/$1/releases/latest" |
    grep '"tag_name":' |
    sed -E 's/.*"([^"]+)".*/\1/'
}

info() {
  echo -e "[\e[93mINFO\e[0m] $1"
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
  if [ -z "$CLUSTER_CONFIG" ]; then
    CLUSTER_CONFIG="cluster/three-nodes-cluster.yaml"
  fi

  info "Using container runtime: $CONTAINER_RUNTIME"
}

create_registry() {
  info "Checking if registry exists..."
  reg_name='kind-registry'
  reg_port='5000'
  running="$(${CONTAINER_RUNTIME} inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
  if [ "${running}" != 'true' ]; then
    info "Registry does not exist, creating..."
    "$CONTAINER_RUNTIME" rm "${reg_name}" 2>/dev/null || true
    "$CONTAINER_RUNTIME" run \
      -d \
      --restart=always \
      -p "${reg_port}:5000" \
      --name "${reg_name}" \
      registry:2
    info "Registry started..."
  else
    info "Registry exists..."
  fi
}

load_config() {
  local config
  config=$(sed "s/\${reg_name}/${reg_name}/g; s/\${reg_port}/${reg_port}/g" "$CLUSTER_CONFIG")
  echo "$config"
}

create_cluster() {
  info "Checking if cluster exists..."
  running_cluster=$(kind get clusters | grep "$KIND_CLUSTER_NAME" || true)
  if [ "${running_cluster}" != "$KIND_CLUSTER_NAME" ]; then
    info "Cluster does not exist, creating with the local registry enabled in containerd..."
    kind create cluster --config=<(load_config)
    info "Waiting for the nodes to be ready..."
    kubectl wait --for=condition=ready node --all --timeout=600s
  else
    info "Cluster exists..."
  fi
}

connect_registry() {
  info "Check if registry is connected to the cluster network..."
  connected_registry=$("$CONTAINER_RUNTIME" network inspect kind -f '{{json .Containers}}' | grep -q "${reg_name}" && echo "true" || echo "false")
  if [ "${connected_registry}" != 'true' ]; then
    info "Registry is not connected, connecting the registry to the cluster network..."
    "$CONTAINER_RUNTIME" network connect "kind" "${reg_name}" || true
    info "Connection established..."
  else
    info "Registry is connected..."
  fi
}

install_tekton() {
  info "Checking if Tekton is installed in the cluster..."
  running_tekton=$(kubectl get crds | grep -q "pipelines.tekton.dev" && echo "true" || echo "false")
  if [ "${running_tekton}" != 'true' ]; then  
    info "Tekton is not installed, installing Tekton Pipeline, Triggers and Dashboard..."
    kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/previous/"${TEKTON_PIPELINE_VERSION}"/release.yaml
    kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/previous/"${TEKTON_TRIGGERS_VERSION}"/release.yaml
    kubectl wait --for=condition=Established --timeout=30s crds/clusterinterceptors.triggers.tekton.dev || true # Starting from triggers v0.13
    kubectl apply -f https://storage.googleapis.com/tekton-releases/triggers/previous/"${TEKTON_TRIGGERS_VERSION}"/interceptors.yaml || true
    kubectl apply -f https://storage.googleapis.com/tekton-releases/dashboard/previous/"${TEKTON_DASHBOARD_VERSION}"/release-full.yaml

    info "Wait until all pods are ready..."
    kubectl wait -n tekton-pipelines --for=condition=ready pods --all --timeout=600s
    kubectl port-forward service/tekton-dashboard -n tekton-pipelines 9097:9097 &>kind-tekton-dashboard.log &
  else
    info "Tekton is installed..."
  fi
  info "Tekton Dashboard available at http://localhost:9097"
}

while getopts ":c:p:t:d:s" opt; do
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
  s)
    SKIP_TEKTON_INSTALL=true
    ;;
  \?)
    echo "Invalid option: $OPTARG" 1>&2
    echo 1>&2
    echo "Usage: tk8.sh [-c cluster-name -p pipeline-version -t triggers-version -d dashboard-version]"
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

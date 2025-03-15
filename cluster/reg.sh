#!/usr/bin/env bash
set -e -o pipefail

info() {
  echo -e "[\e[93mINFO\e[0m] $1"
}

create_local_registry() {
  reg_name='test-registry'
  reg_port='5001'
  info "Checking if '$reg_name' exists..."
  running="$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)"
  if [ "${running}" != 'true' ]; then
    info "Registry '$reg_name' does not exist, creating..."
    # It may exists and not be running, so cleanup just in case
    docker rm "${reg_name}" 2>/dev/null || true
    # And start a new one
    docker run \
      -d \
      --restart=always \
      --name "${reg_name}" \
      -p "${reg_port}:5000" \
      -v ./data/local:/var/lib/registry:z \
      registry:2
    info "Registry '$reg_name' started..."
  else
    info "Registry '$reg_name' exists..."
  fi
}

create_local_registry

#!/usr/bin/env bash
set -e -o pipefail

info() {
  echo -e "[\e[93mINFO\e[0m] $1"
}

get_latest_release() {
  curl --silent "https://api.github.com/repos/$1/releases/latest" |
    grep '"tag_name":' |
    sed -E 's/.*"([^"]+)".*/\1/'
}

TEKTON_PIPELINE_VERSION=$(get_latest_release tektoncd/pipeline)
TEKTON_TRIGGERS_VERSION=$(get_latest_release tektoncd/triggers)
TEKTON_DASHBOARD_VERSION=$(get_latest_release tektoncd/dashboard)

info "latest pipeline version: $TEKTON_PIPELINE_VERSION"
info "latest triggers version: $TEKTON_TRIGGERS_VERSION"
info "latest dashboard version: $TEKTON_DASHBOARD_VERSION"
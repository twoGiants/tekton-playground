#!/usr/bin/env bash
set -e -o pipefail

function up {
  kubectl apply -k getting-started/pipeline
  kubectl apply -k getting-started/triggers
}

function down {
  kubectl delete -k getting-started/pipeline
  kubectl delete -k getting-started/triggers
}

function usage {
  echo "Usage: $0 {up|down}"
}

if [ $# -eq 0 ]; then
  usage "$0"
  exit 1
fi

case $1 in
  up)
    up
    ;;
  down)
    down
    ;;
  *)
    echo "Invalid argument: $1"
    usage "$0"
    ;;
esac
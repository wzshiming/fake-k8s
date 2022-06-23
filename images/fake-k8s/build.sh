#!/usr/bin/env bash

DIR="$(dirname "${BASH_SOURCE[@]}")"

image="${1}"
if [[ -z "${image}" ]]; then
  echo "Usage: ${0} <image>"
  exit 1
fi

docker buildx build \
  --push \
  --platform linux/amd64,linux/arm64 \
  -t "${image}" \
  -f "${DIR}/Dockerfile" \
  "${DIR}/../.."

#!/usr/bin/env bash

DIR="$(dirname "${BASH_SOURCE[@]}")"

base_image="${1}"
cluster_image_prefix="${2}"

if [[ -z "${base_image}" ]] || [[ -z "${cluster_image_prefix}" ]]; then
  echo "Usage: ${0} <base_image> <cluster_image_prefix}"
  exit 1
fi

for release in $(cat "${DIR}/../../supported_releases.txt"); do
  docker buildx build \
    --push \
    --platform linux/amd64,linux/arm64 \
    --build-arg base_image="${base_image}" \
    --build-arg kube_version="${release}" \
    -t "${cluster_image_prefix}:${release%\.*}" \
    -f "${DIR}/Dockerfile" \
    "${DIR}"
done

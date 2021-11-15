#!/bin/bash

VERSION="${VERSION:-main.1907}" # Build ID taken from https://werft.gitpod-dev.com/

# shellcheck disable=SC2068
docker run -it --rm \
  -v "$PWD:/gitpod" \
  -v "$HOME/.kube:/root/.kube" \
  "eu.gcr.io/gitpod-core-dev/build/installer:${VERSION}" \
  $@

#!/bin/bash
cd $(dirname $0)/../network-config-ui

docker run -ti --rm -v .:/work -w /work --network host node:22-slim bash
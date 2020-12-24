#!/bin/bash
set -e

DEPLOY_DIR=/home/david/Documents/media-hub

cd "${DEPLOY_DIR}"

exec srt-server \
	-config=srt.yml

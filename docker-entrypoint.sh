#!/bin/sh
set -eu

mkdir -p /data
chown -R app:app /data
exec su-exec app /usr/local/bin/scan-collector

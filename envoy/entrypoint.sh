#!/bin/sh
set -eu

envsubst < /etc/envoy/envoy.yaml.tmpl > /etc/envoy/envoy.yaml

exec envoy -c /etc/envoy/envoy.yaml --service-cluster user-core-gateway --service-node user-core-gateway
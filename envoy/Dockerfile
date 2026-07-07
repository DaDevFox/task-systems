FROM envoyproxy/envoy:v1.31-latest

USER root

RUN apt-get update \
    && apt-get install -y --no-install-recommends gettext-base \
    && rm -rf /var/lib/apt/lists/*

COPY envoy.yaml.tmpl /etc/envoy/envoy.yaml.tmpl
COPY entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
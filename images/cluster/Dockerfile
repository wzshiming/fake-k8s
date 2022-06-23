ARG base_image
FROM $base_image

ARG kube_version
ENV KUBE_VERSION $kube_version

RUN fake-k8s create cluster --quiet-pull && \
    fake-k8s kubectl version && \
    fake-k8s delete cluster

COPY entrypoint.sh /entrypoint.sh

# Used in entrypoint.sh not in fake-k8s
ENV APISERVER_PORT 8080
ENTRYPOINT [ "/entrypoint.sh" ]

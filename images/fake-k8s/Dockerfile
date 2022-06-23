FROM golang:alpine AS builder
WORKDIR /go/src/github.com/wzshiming/fake-k8s/
COPY . .
ENV CGO_ENABLED=0
RUN go install ./cmd/fake-k8s

FROM alpine
COPY --from=builder /go/bin/fake-k8s /usr/local/bin/
ENTRYPOINT [ "/usr/local/bin/fake-k8s" ]

FROM docker.io/library/golang:1.11.1 as builder
ADD . /work
WORKDIR /work
RUN CGO_ENABLED=0 GOOS=linux go build -o container-image-lister

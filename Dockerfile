FROM golang:1.16 AS build-env

WORKDIR /app
COPY . .

RUN go mod download

ENV GO111MODULE=on
ENV CGO_ENABLED=0

RUN go build -o pod-time-controller cmd/main.go && chmod 555 pod-time-controller

FROM scratch
# Alternatively if one needs a shell for debugging...
#FROM gcr.io/distroless/base-debian10:debug

COPY --from=build-env /app/pod-time-controller /
USER 10001
CMD ["/pod-time-controller"]
ENTRYPOINT ["/pod-time-controller"]


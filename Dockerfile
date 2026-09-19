# Сборка и прогон вашего решения.
#   docker build -t pp-lab-go .
#   docker run --rm pp-lab-go info
FROM golang:1.24-bookworm AS build

WORKDIR /src
COPY . .
RUN go vet ./... \
 && go test ./... \
 && CGO_ENABLED=0 go build -o /out/harness ./cmd/harness

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/harness /harness
ENTRYPOINT ["/harness"]
CMD ["info"]

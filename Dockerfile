FROM --platform=$BUILDPLATFORM golang:1.25.5 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

# Copy only the source needed to build the binary (no recursive context copy).
COPY interfaces.go server.go types.go ./
COPY cluster/ cluster/
COPY tools/ tools/
COPY cmd/ cmd/

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
  -trimpath \
  -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
  -o /kai ./cmd/kai

FROM gcr.io/distroless/static:nonroot

EXPOSE 8080

USER nonroot:nonroot

COPY --from=build /kai /kai

ENTRYPOINT [ "/kai" ]

# See: https://github.com/chemidy/smallest-secured-golang-docker-image

ARG BUILDER_IMAGE=golang:alpine

FROM ${BUILDER_IMAGE} AS builder

# Install git + SSL ca certificates.
# git is required for fetching the dependencies.
# ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates

# Create the user that the app will run as
ENV USER=appuser
ENV UID=10001

# See https://stackoverflow.com/a/55757473/12429735
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"
WORKDIR $GOPATH/src/mypackage/myapp/
COPY . .

# Fetch dependencies.
ENV GIN_MODE=release
RUN go get -d -v

RUN \
  CGO_ENABLED=0 \
  GOOS=linux \
  GOARCH=amd64 \
  go build \
  -ldflags='-w -s -extldflags "-static"' \
  -a \
  -o /go/bin/landlord .

FROM scratch

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy our static executable
COPY --from=builder /go/bin/landlord /go/bin/landlord

USER appuser:appuser

ENV GIN_MODE=release

ENTRYPOINT ["/go/bin/landlord"]

FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /nutmetrics .

FROM scratch
COPY --from=build /nutmetrics /nutmetrics
USER 65532:65532
EXPOSE 9199
ENTRYPOINT ["/nutmetrics"]

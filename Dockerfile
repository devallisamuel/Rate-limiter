# ---- Stage 1: build ----
FROM golang:1.27 AS build
WORKDIR /src

# Copy go.mod first so dependency downloads are cached between builds.
COPY go.mod ./
RUN go mod download

COPY . .
# CGO off + static binary so it runs in a minimal image with no libc.
RUN CGO_ENABLED=0 go build -o /rate-limiter ./server

# ---- Stage 2: run ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /rate-limiter /rate-limiter
EXPOSE 8080
ENTRYPOINT ["/rate-limiter"]

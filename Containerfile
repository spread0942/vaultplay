# syntax=docker/dockerfile:1.7

FROM docker.io/library/node:22-bookworm AS frontend
WORKDIR /src/web
COPY web/package.json ./
RUN npm install
COPY web/ ./
RUN npm run build

FROM docker.io/library/golang:1.24-bookworm AS backend
WORKDIR /src
COPY go.mod ./
COPY main.go ./
COPY internal ./internal
COPY --from=frontend /src/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /vaultplay .

FROM gcr.io/distroless/static-debian12
COPY --from=backend /vaultplay /vaultplay
ENTRYPOINT ["/vaultplay"]

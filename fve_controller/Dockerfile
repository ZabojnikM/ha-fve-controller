FROM node:24.19.0-alpine3.24@sha256:d32cdf619f63fe0471182d08996dd516c6275bb5fd31ae06e55a570bd9e1ad43 AS frontend
WORKDIR /src/web
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN corepack enable && corepack prepare pnpm@11.25.0 --activate && pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM golang:1.27.1-alpine3.24@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go test ./... && go vet ./... && CGO_ENABLED=0 go build -trimpath -o /controller ./cmd/controller

FROM alpine:3.22.3@sha256:55ae5d250caebc548793f321534bc6a8ef1d116f334f18f4ada1b2daad3251b2
ARG VERSION=0.1.1
ARG REVISION=local
LABEL org.opencontainers.image.title="FVE Controller" \
      org.opencontainers.image.source="https://github.com/ZabojnikM/ha-fve-controller" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.description="Observe-only FVE simulator for Home Assistant"
WORKDIR /app
COPY --from=backend /controller /app/controller
COPY --from=frontend /src/web/dist /app/web
ENV LISTEN_ADDR=0.0.0.0:8099 DATA_DIR=/data WEB_DIR=/app/web INGRESS_ONLY=true
EXPOSE 8099
CMD ["/app/controller"]

# --- Build stage -----------------------------------------------------------
FROM golang:1.26.4 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /auth-service .

# --- Run stage -------------------------------------------------------------
FROM gcr.io/distroless/static-debian12
COPY --from=build /auth-service /auth-service
EXPOSE 8000
ENTRYPOINT ["/auth-service"]

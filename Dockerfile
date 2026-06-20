FROM golang:1.26.4

RUN apt-get update && apt-get install -y \
    tesseract-ocr \
    libtesseract-dev \
    libleptonica-dev \
    pkg-config

WORKDIR /build

COPY go.mod go.sum .
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o /app ./cmd/fin-track

ENTRYPOINT ["/app"]

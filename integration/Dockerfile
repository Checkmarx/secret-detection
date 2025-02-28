FROM golang:1.23.6

WORKDIR /app

RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
 && rm -rf /var/lib/apt/lists/*

RUN pip3 install --break-system-packages pre-commit

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN chmod +x integration/bin/cx

RUN go test ./integration/...


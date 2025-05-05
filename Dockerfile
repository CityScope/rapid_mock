FROM golang:1.20-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Create the required data directories if they don't exist
RUN mkdir -p data/a data/b

# Build the application
RUN go build -o rapid_mock .

# Expose port 8080
EXPOSE 8080

# Define volume for media files
VOLUME ["/app/data"]

CMD ["./rapid_mock"]
FROM golang:1.16-alpine AS builder

ARG APP_MODULES=all

# Move to working directory (/build).
WORKDIR /build

# Copy and download dependency using go mod.
COPY go.mod go.sum ./
RUN go mod download

# Copy the code into the container.
COPY . .

# Set necessary environmet variables needed for our image and build the API server.
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -tags ${APP_MODULES} -ldflags="-s -w" -o nervatura .

FROM scratch

# Copy binary and config files from /build to root folder of scratch container.
COPY --from=builder ["/build/nervatura", "/build/.env", "/"]
COPY --from=builder "/build/static" "/static"

# Export necessary port.
EXPOSE 5000

# Command to run when starting the container.
ENTRYPOINT ["/nervatura"]

# This Dockerfile builds and runs a Go-based game server called Cardinal.
# 
# The build stage uses the official Golang 1.19 image to compile the application.
# - Sets the working directory to /Cardinal.
# - Copies the necessary source files and dependencies.
# - Downloads Go modules and builds the application with CGO disabled for Linux.
# 
# The runner stage uses an Alpine 3.11 image to run the compiled application.
# - Sets environment variables for timezone, user ID, and group ID.
# - Installs bash, curl, and glibc on Alpine.
# - Sets the working directory to /Cardinal and creates a configuration directory.
# - Copies the compiled application and locales from the build stage.
# - Grants execution permissions to the application binary.
# 
# The container exposes port 19999 and runs the Cardinal application.
FROM golang:1.19 AS builder

WORKDIR /Cardinal
COPY internal internal
COPY locales locales
COPY cardinal.go cardinal.go
COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /Cardinal

FROM alpine:3.11 AS runner

# Set environment variables
ENV TZ="Europe/Amsterdam"
ENV PUID=19999
ENV PGID=19999

# Install glibc on Alpine
RUN apk add --no-cache bash curl \
    && curl -Lo /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub \
    && curl -Lo glibc.apk https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.34-r0/glibc-2.34-r0.apk \
    && apk add glibc.apk \
    && rm glibc.apk

RUN mkdir /checker

WORKDIR /Cardinal
RUN mkdir /Cardinal/conf

COPY --from=builder /Cardinal/Cardinal /Cardinal/Cardinal
COPY ./locales /Cardinal/locales

RUN chmod +x /Cardinal/Cardinal

EXPOSE 19999
CMD ["/Cardinal/Cardinal"]
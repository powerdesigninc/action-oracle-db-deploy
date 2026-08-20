# syntax=docker/dockerfile:1

ARG GO_VERSION=1.24
FROM golang:${GO_VERSION}-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY src/ ./src/
RUN CGO_ENABLED=0 go build -o /out/oracle-db-deploy ./src

FROM ubuntu:24.04

# Stable "latest" link from Oracle; override to pin a specific release.
ARG SQLCL_URL=https://download.oracle.com/otn_software/java/sqldeveloper/sqlcl-latest.zip

ENV DEBIAN_FRONTEND=noninteractive

# openconnect + vpnc-scripts/iproute2 for the VPN step,
# JRE for SQLcl, git/curl/unzip for checkout and installs.
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        git \
        iproute2 \
        openconnect \
        openjdk-17-jre-headless \
        unzip \
        vpnc-scripts \
    && rm -rf /var/lib/apt/lists/*

# SQLcl
RUN curl -fsSL "$SQLCL_URL" -o /tmp/sqlcl.zip \
    && unzip -q /tmp/sqlcl.zip -d /opt \
    && rm /tmp/sqlcl.zip \
    && ln -s /opt/sqlcl/bin/sql /usr/local/bin/sql

COPY --from=build /out/oracle-db-deploy /usr/local/bin/oracle-db-deploy

WORKDIR /workspace

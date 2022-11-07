FROM golang:1.16-alpine  AS base

# add git
RUN apk add git

# Copy source for AirBrake code hunter
COPY pkg /src/pkg

# Copy command scripts & neccessary files
COPY bin/cmd /
WORKDIR /

FROM base AS rpc
CMD ["/rpc"]

FROM base AS cron
CMD ["/cron"]

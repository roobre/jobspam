FROM golang:1.21-alpine3.18 as builder

WORKDIR /jobspam
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache \
  go build -o jobspam .

FROM alpine:3.18
COPY --from=builder /jobspam/jobspam /usr/local/bin/jobspam
ENTRYPOINT [ "/usr/local/bin/jobspam" ]

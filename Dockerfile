FROM efrecon/s3fs:1.86 as bin-s3fs
FROM rclone/rclone:1.53  as bin-rclone

FROM golang:1.15-alpine as builder
RUN apk add make binutils
COPY / /work
WORKDIR /work
RUN make

FROM alpine:3.12
RUN apk --no-cache add \
    ca-certificates \
    fuse \
    libxml2 \
    libcurl \
    libgcc \
    libstdc++
COPY --from=bin-s3fs /usr/bin/s3fs /usr/bin/s3fs
RUN /usr/bin/s3fs --version
COPY --from=bin-rclone /usr/local/bin/rclone /usr/local/bin/rclone
RUN /usr/local/bin/rclone --version
COPY --from=builder /work/bin/s3driver /s3driver
ENTRYPOINT ["/s3driver"]

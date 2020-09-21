FROM alpine:latest
RUN set -x && \
    addgroup -S depoy && adduser -S -G depoy depoy && \
    mkdir -p  /home/depoy/data && \
    chown -R depoy:depoy /home/depoy

USER depoy
WORKDIR /home/depoy
COPY ./depoy ./
VOLUME /home/depoy/data
CMD ["./depoy"]

FROM node AS vueBuilder
ENV NODE_ENV=production
ARG HTTP_PROXY
ARG HTTPS_PROXY
WORKDIR /usr/src/app
COPY vue ./
RUN npm install
RUN npm run build

FROM golang:1.15 AS goBuilder
ARG HTTP_PROXY
ARG HTTPS_PROXY
WORKDIR /go/src/app
COPY depoy ./
COPY --from=vueBuilder /usr/src/app/dist ../vue/dist
RUN go get -u github.com/gobuffalo/packr/packr && \
    CGO_ENABLED=0 GOARCH=amd64 GOOS=linux packr build -a -o depoy .


FROM alpine:latest  
RUN set -x && \
    addgroup -S depoy && adduser -S -G depoy depoy && \
    mkdir -p  /home/depoy/data && \
    chown -R depoy:depoy /home/depoy

USER depoy
WORKDIR /home/depoy
COPY --from=goBuilder /go/src/app/depoy ./
VOLUME /home/depoy/data
CMD ["./depoy"]
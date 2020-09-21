FROM node AS vueBuilder
ENV NODE_ENV=production
ARG HTTP_PROXY
ARG HTTPS_PROXY
WORKDIR /usr/src/app
COPY webapp ./
RUN npm install
RUN npm list -g --depth 0
RUN npm run build

FROM golang:1.15 AS goBuilder
ARG HTTP_PROXY
ARG HTTPS_PROXY
WORKDIR /go/src/app
COPY ./ ./
COPY --from=vueBuilder /usr/src/app/dist ../webapp/dist
RUN go get -u github.com/gobuffalo/packr/packr
RUN CGO_ENABLED=0 GOARCH=amd64 GOOS=linux packr build -a -o depoy .

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

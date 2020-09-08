FROM node AS vueBuilder
ENV NODE_ENV=dev
WORKDIR /usr/src/app
COPY vue ./
RUN npm install
RUN npm run build

FROM golang:1.14 AS goBuilder
WORKDIR /go/src/app
COPY depoy ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM alpine:latest  
WORKDIR /root/
COPY --from=vueBuilder /usr/src/app/dist ./public
COPY --from=goBuilder /go/src/app/app ./
CMD ["./app"]
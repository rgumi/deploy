# Depoy

[![Build Status](https://travis-ci.com/rgumi/depoy.svg?branch=master)](https://travis-ci.com/rgumi/depoy)

Depoy is an API-Gateway which natively supports Continous Deployment (CD) of RESTful-Application. It evaluates the state of an upstream application by collecting HTTP-Connection metrics and by scraping the Prometheus-Endpoint of the upstream application - if provided. It integrates into Prometheus and offers a reactive web-application for configuration and monitoring.

```{r, out.width="0.3\\linewidth", include=TRUE, fig.align="center", fig.cap=c("APIGatewayOverview.pdf"), echo=FALSE}
knitr::include_graphics("https://github.com/rgumi/depoy/raw/master/images/APIGatewayOverview.pdf")
```

## Architecture

The API-Gateway is built using Go for all backend tasks and Vue for the web-application.

```{r, out.width="0.3\\linewidth", include=TRUE, fig.align="center", fig.cap=c("OverviewDiagram.pdf"), echo=FALSE}
knitr::include_graphics("https://github.com/rgumi/depoy/raw/master/images/OverviewDiagram.pdf")
```

## Building

Using the provided ["Dockerfile_multistage"](Dockerfile_multistage) you are able to build the dockerimage yourself. A prebuild image can be found [here](https://hub.docker.com/repository/docker/rgummich/depoy).

By using npm and go it is also possible to build the executable without needing Docker.

```lang-bash
cd webapp
npm install
npm run build
cd ..
go get -u github.com/gobuffalo/packr/v2/packr2
CGO_ENABLED=0 packr2 build -a -o depoy .
```

## Deployment

Depoy provides args that can be used to configure the core components. Using "./depoy --help" you are able to view all args and their default values.
When starting Depoy these args can be set, e. g. through Dockers entrypoint.

## Examples

Examples of configurations in YAML can be found under the folder "examples".

## Access 
package statemgt

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rgumi/depoy/config"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

func (s *StateMgt) HealthzHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(200)
	ctx.SetBody([]byte("{\"status\": \"ok\"}"))
}

func (s *StateMgt) GetCurrentConfig(ctx *fasthttp.RequestCtx) {
	cfg, err := s.Gateway.ReadConfig()
	if err != nil {
		returnError(ctx, 500, err, nil)
		return
	}
	marshalAndReturn(ctx, cfg)
}

func (s *StateMgt) SetCurrentConfig(ctx *fasthttp.RequestCtx) {

	if string(ctx.Request.Header.ContentType()) != "application/json" {
		returnError(ctx, 400, fmt.Errorf("Content-Type must be application/json"), nil)
		return
	}

	b := ctx.Request.Body()
	newGateway, err := config.ParseFromBinary(json.Unmarshal, b)
	if err != nil {
		log.Error(err)
		returnError(ctx, 400, err, nil)
		return
	}

	ctx.SetStatusCode(201)

	go func() {
		s.Gateway.Stop()
		time.Sleep(2 * time.Second)
		s.Gateway = newGateway
		go s.Gateway.Run()
	}()
}

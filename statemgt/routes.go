package statemgt

import (
	"fmt"

	"github.com/rgumi/depoy/config"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"

	"github.com/google/uuid"
)

/*
	Routes
*/

// GetRouteByName returns the route with given name
func (s *StateMgt) GetRouteByName(ctx *fasthttp.RequestCtx) {
	name := string(ctx.QueryArgs().Peek("name"))
	route := s.Gateway.GetRoute(name)
	if route == nil {
		s.GetAllRoutes(ctx)
		return
	}
	marshalAndReturn(ctx, config.ConvertRouteToInputRoute(route))
}

// GetAllRoutes returns all defined routes of the Gateway
func (s *StateMgt) GetAllRoutes(ctx *fasthttp.RequestCtx) {

	routes := s.Gateway.GetRoutes()
	if routes == nil {
		ctx.SetStatusCode(404)
		return
	}
	output := make(map[string]*config.InputRoute, len(routes))
	for idx, route := range routes {
		output[idx] = config.ConvertRouteToInputRoute(route)
	}
	marshalAndReturn(ctx, output)
}

// CreateRoute creates a new Route. If route already exist, error
func (s *StateMgt) CreateRoute(ctx *fasthttp.RequestCtx) {
	myRoute := config.NewInputRoute()
	if err := readBodyAndUnmarshal(ctx, myRoute); err != nil {
		returnError(ctx, 400, err, nil)
		return
	}

	newRoute, err := config.ConvertInputRouteToRoute(myRoute)
	if err != nil {
		returnError(ctx, 400, err, nil)
		return
	}
	if err = myRoute.Strategy.Validate(newRoute); err != nil {
		returnError(ctx, 400, err, nil)
		return
	}
	err = myRoute.Strategy.Reset(newRoute)
	if err != nil {
		returnError(ctx, 400, err, nil)
		return
	}

	if err = s.Gateway.RegisterRoute(newRoute); err != nil {
		returnError(ctx, 400, err, nil)
		return
	}

	newRoute.Reload()
	s.Gateway.Reload()
	marshalAndReturn(ctx, config.ConvertRouteToInputRoute(newRoute))
}

// DeleteRouteByName removed the given route and all its backends
func (s *StateMgt) DeleteRouteByName(ctx *fasthttp.RequestCtx) {
	name := string(ctx.QueryArgs().Peek("name"))
	route := s.Gateway.RemoveRoute(name)
	if route == nil {
		ctx.SetStatusCode(404)
		return
	}
	marshalAndReturn(ctx, config.ConvertRouteToInputRoute(route))
}

// UpdateRouteByName removed route and replaces it with new route
func (s *StateMgt) UpdateRouteByName(ctx *fasthttp.RequestCtx) {
	myRoute := config.NewInputRoute()
	routeName := string(ctx.QueryArgs().Peek("name"))

	if err := readBodyAndUnmarshal(ctx, myRoute); err != nil {
		returnError(ctx, 400, err, nil)
		return
	}

	// Both routes need to have the same name otherwise the old one cant be replaced
	// CreateRoute should be used when creating a new Route
	if myRoute.Name != routeName {
		returnError(ctx, 400, fmt.Errorf("Names must be equal. Otherwise they cant be replaced"), nil)
		return
	}
	newRoute, err := config.ConvertInputRouteToRoute(myRoute)
	if err != nil {
		returnError(ctx, 400, err, nil)
		return
	}

	if err = myRoute.Strategy.Validate(newRoute); err != nil {
		returnError(ctx, 400, err, nil)
		return
	}
	err = myRoute.Strategy.Reset(newRoute)
	if err != nil {
		returnError(ctx, 400, err, nil)
		return
	}
	// remove first
	s.Gateway.RemoveRoute(newRoute.Name)
	if err = s.Gateway.RegisterRoute(newRoute); err != nil {
		returnError(ctx, 500, err, nil)
		return
	}

	newRoute.Reload()
	s.Gateway.Reload()
	marshalAndReturn(ctx, config.ConvertRouteToInputRoute(newRoute))
}

/*
	Backends
*/

// AddNewBackendToRoute adds a new backend to the defined route
func (s *StateMgt) AddNewBackendToRoute(ctx *fasthttp.RequestCtx) {
	myBackend := config.NewInputBackend()
	routeName := string(ctx.QueryArgs().Peek("route"))
	route, found := s.Gateway.Routes[routeName]
	if !found {
		returnError(ctx, 404, fmt.Errorf("Could not find route"), nil)
		return
	}
	if err := readBodyAndUnmarshal(ctx, myBackend); err != nil {
		returnError(ctx, 400, err, nil)
		return
	}
	for _, cond := range myBackend.Metricthresholds {
		cond.Compile()
	}
	newBackend, err := config.ConvertInputBackendToBackend(myBackend)
	if err != nil {
		returnError(ctx, 400, err, nil)
		return
	}
	if _, err = route.AddExistingBackend(newBackend); err != nil {
		returnError(ctx, 400, err, nil)
		return
	}

	route.Reload()
	log.Debug("Sucessfully updated route")
	marshalAndReturn(ctx, config.ConvertRouteToInputRoute(route))
}

// RemoveBackendFromRoute remoes a backend from the defined route
// if route is not found, returns 404
func (s *StateMgt) RemoveBackendFromRoute(ctx *fasthttp.RequestCtx) {
	routeName := string(ctx.QueryArgs().Peek("route"))
	id := string(ctx.QueryArgs().Peek("backend"))

	backendID, err := uuid.Parse(id)
	if err != nil {
		returnError(ctx, 400, fmt.Errorf("Invalid uuid for backendID"), nil)
		return
	}

	route, found := s.Gateway.Routes[routeName]
	if !found {
		returnError(ctx, 404, fmt.Errorf("Could not find route"), nil)
		return
	}
	if err = route.RemoveBackend(backendID); err != nil {
		returnError(ctx, 400, err, nil)
		return
	}
	marshalAndReturn(ctx, config.ConvertRouteToInputRoute(route))
}

/*
	Switchover
*/

// CreateSwitchover adds a switchover struct to the given route
func (s *StateMgt) CreateSwitchover(ctx *fasthttp.RequestCtx) {
	mySwitchOver := config.NewInputSwitchover()
	routeName := string(ctx.QueryArgs().Peek("route"))
	route, found := s.Gateway.Routes[routeName]
	if !found {
		returnError(ctx, 404, fmt.Errorf("Could not find route"), nil)
		return
	}
	if err := readBodyAndUnmarshal(ctx, mySwitchOver); err != nil {
		returnError(ctx, 400, err, nil)
		return
	}

	newSwitchover, err := route.StartSwitchOver(
		mySwitchOver.From,
		mySwitchOver.To,
		mySwitchOver.Conditions,
		mySwitchOver.Timeout.Duration,
		mySwitchOver.AllowedFailures,
		mySwitchOver.WeightChange,
		mySwitchOver.Force,
		mySwitchOver.Rollback,
	)
	if err != nil {
		returnError(ctx, 400, err, nil)
		return
	}
	marshalAndReturn(ctx, config.ConvertSwitchoverToInputSwitchover(newSwitchover))

}

// GetSwitchover returns the state of the current switchover of the given route
func (s *StateMgt) GetSwitchover(ctx *fasthttp.RequestCtx) {
	routeName := string(ctx.QueryArgs().Peek("route"))

	route, found := s.Gateway.Routes[routeName]
	if !found {
		returnError(ctx, 404, fmt.Errorf("Could not find route"), nil)
		return
	}

	if route.Switchover == nil {
		returnError(ctx, 404, fmt.Errorf("Route does not have a swtichover active"), nil)
		return
	}
	marshalAndReturn(ctx, config.ConvertSwitchoverToInputSwitchover(route.Switchover))
}

// DeleteSwitchover stops and removes the switchover of the given route
// if no switchover is active, 404 is returned
func (s *StateMgt) DeleteSwitchover(ctx *fasthttp.RequestCtx) {
	routeName := string(ctx.QueryArgs().Peek("route"))

	route, found := s.Gateway.Routes[routeName]
	if !found {
		returnError(ctx, 404, fmt.Errorf("Could not find route"), nil)
		return
	}

	if route.Switchover == nil {
		returnError(ctx, 404, fmt.Errorf("Route does not have a swtichover active"), nil)
		return
	}
	route.RemoveSwitchOver()
	ctx.SetStatusCode(200)
}

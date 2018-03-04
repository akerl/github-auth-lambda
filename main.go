package main

import (
	"fmt"
	"regexp"

	"github.com/akerl/go-lambda/apigw/events"
	"github.com/akerl/go-lambda/apigw/router"
)

var debugRegex = regexp.MustCompile(`^/debug$`)
var authRegex = regexp.MustCompile(`^/auth$`)
var callbackRegex = regexp.MustCompile(`^/callback$`)
var indexRegex = regexp.MustCompile(`^/$`)
var defaultRegex = regexp.MustCompile(`.*`)

func debugHandler(req events.Request) (events.Response, error) {
	return events.Succeed(fmt.Sprintf("%+v\n", req))
}

func authHandler(req events.Request) (events.Response, error) {
	return events.Succeed("Auth!")
}

func callbackHandler(req events.Request) (events.Response, error) {
	return events.Succeed("Callback!")
}

func indexHandler(req events.Request) (events.Response, error) {
	return events.Succeed("Index!")
}

func defaultHandler(req events.Request) (events.Response, error) {
	return events.Succeed(fmt.Sprintf("Catch-all: %s", req.Path))
}

func main() {
	r := router.Router{
		Routes: []router.Route{
			router.Route{Path: debugRegex, Handler: debugHandler},
			router.Route{Path: authRegex, Handler: authHandler},
			router.Route{Path: callbackRegex, Handler: callbackHandler},
			router.Route{Path: indexRegex, Handler: indexHandler},
			router.Route{Path: defaultRegex, Handler: defaultHandler},
		},
	}
	r.Start()
}

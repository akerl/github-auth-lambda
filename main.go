package main

import (
	"regexp"

	"github.com/akerl/go-lambda/apigw/router"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var (
	config   *configFile
	sm       *sessionManager
	oauthCfg *oauth2.Config
	scopes   = []string{"read:org"}

	authRegex     = regexp.MustCompile(`^/auth$`)
	callbackRegex = regexp.MustCompile(`^/callback$`)
	indexRegex    = regexp.MustCompile(`^/$`)
	faviconRegex  = regexp.MustCompile(`/favicon.ico`)
	defaultRegex  = regexp.MustCompile(`/.*`)
)

func main() {
	var err error

	config, err = loadConfig()
	if err != nil {
		panic(err)
	}

	sm = &sessionManager{
		Name:     "session",
		SignKey:  config.SignKey,
		EncKey:   config.EncKey,
		Lifetime: config.Lifetime,
		Domain:   config.Domain,
	}

	oauthCfg = &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Endpoint:     github.Endpoint,
		Scopes:       scopes,
	}

	r := router.Router{
		Routes: []router.Route{
			router.Route{Path: authRegex, Handler: authHandler},
			router.Route{Path: callbackRegex, Handler: callbackHandler},
			router.Route{Path: indexRegex, Handler: indexHandler},
			router.Route{Path: faviconRegex, Handler: faviconHandler},
			router.Route{Path: defaultRegex, Handler: defaultHandler},
		},
	}
	r.Start()
}

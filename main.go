package main

import (
	"regexp"

	"github.com/akerl/github-auth-lambda/session"

	"github.com/akerl/go-lambda/apigw/router"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var (
	config   *configFile
	sm       *session.Manager
	oauthCfg *oauth2.Config
	scopes   = []string{"read:org"}

	authRegex     = regexp.MustCompile(`^/auth$`)
	logoutRegex   = regexp.MustCompile(`^/logout$`)
	callbackRegex = regexp.MustCompile(`^/callback$`)
	indexRegex    = regexp.MustCompile(`^/$`)
	faviconRegex  = regexp.MustCompile(`^/favicon.ico$`)
	defaultRegex  = regexp.MustCompile(`^/.*$`)
)

func main() {
	var err error

	config, err = loadConfig()
	if err != nil {
		panic(err)
	}

	sm = &session.Manager{
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
			{Path: authRegex, Handler: authHandler},
			{Path: logoutRegex, Handler: logoutHandler},
			{Path: callbackRegex, Handler: callbackHandler},
			{Path: indexRegex, Handler: indexHandler},
			{Path: faviconRegex, Handler: faviconHandler},
			{Path: defaultRegex, Handler: defaultHandler},
		},
	}
	r.Start()
}

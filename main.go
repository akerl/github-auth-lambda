package main

import (
	"regexp"

	"github.com/akerl/github-auth-lambda/session"
	"github.com/akerl/go-lambda/mux"
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

	d := mux.NewDispatcher(
		mux.NewRoute(authRegex, authHandler),
		mux.NewRoute(logoutRegex, logoutHandler),
		mux.NewRoute(callbackRegex, callbackHandler),
		mux.NewRoute(indexRegex, indexHandler),
		mux.NewRoute(faviconRegex, faviconHandler),
		mux.NewRoute(defaultRegex, defaultHandler),
	)
	mux.Start(d)
}

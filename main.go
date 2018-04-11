package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"regexp"

	"github.com/akerl/go-lambda/apigw/events"
	"github.com/akerl/go-lambda/apigw/router"
	"github.com/google/go-github/github"
	"github.com/gorilla/securecookie"
	"golang.org/x/oauth2"
	githubOauth "golang.org/x/oauth2/github"
)

// TODO: Clean up error messages / logging
// TODO: Return useful HTTP error codes for failure
// TODO: Review error leakage to front/backends
// TODO: Handle favicon

const (
	redirectURL = ""
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

func authHandler(req events.Request) (events.Response, error) {
	// TODO: Don't auth if already valid creds
	b := make([]byte, 16)
	rand.Read(b)
	nonce := base64.URLEncoding.EncodeToString(b)

	sess, err := sm.Read(req)
	if err != nil {
		log.Printf("Failed loading session cookie: %s", err)
		return events.Fail("error; aborting")
	}

	sess.Nonce = nonce
	cookie, err := sm.Write(sess)
	if err != nil {
		log.Printf("Failed writing session cookie: %s", err)
		return events.Fail("error; aborting")
	}
	url := oauthCfg.AuthCodeURL(nonce)

	return events.Response{
		StatusCode: 303,
		Headers: map[string]string{
			"Location":   url,
			"Set-Cookie": cookie,
		},
	}, nil
}

func callbackHandler(req events.Request) (events.Response, error) {
	// TODO: Break this method apart
	sess, err := sm.Read(req)
	if err != nil {
		log.Printf("Failed loading session cookie: %s", err)
		return events.Fail("error; aborting")
	}

	actual := req.QueryStringParameters["state"]

	if sess.Nonce == "" {
		log.Print("callback hit with no nonce")
		return events.Redirect("https://"+req.Headers["Host"], 303)
	} else if sess.Nonce != actual {
		log.Print("nonce mismatch; possible csrf OR cookies not enabled")
		return events.Fail("error; aborting")
	}

	code := req.QueryStringParameters["code"]
	token, err := oauthCfg.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Printf("there was an issue getting your token: %s", err)
		return events.Fail("error; aborting")
	}

	if !token.Valid() {
		log.Print("retreived invalid token")
		return events.Fail("error; aborting")
	}

	client := github.NewClient(oauthCfg.Client(oauth2.NoContext, token))
	user, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		log.Printf("error getting name: %s", err)
		return events.Fail("error; aborting")
	}
	orgs, _, err := client.Organizations.List(context.Background(), "", &github.ListOptions{})
	if err != nil {
		log.Print("error getting orgs")
		return events.Fail("error; aborting")
	}
	var orgList []string
	for _, i := range orgs {
		orgList = append(orgList, *i.Login)
	}

	sess.Login = *user.Name
	sess.Orgs = orgList

	cookie, err := sm.Write(sess)
	if err != nil {
		log.Print("error encoding cookie")
		return events.Fail("error; aborting")
	}
	return events.Response{
		StatusCode: 303,
		Headers: map[string]string{
			"Location":   "https://" + req.Headers["Host"],
			"Set-Cookie": cookie,
		},
	}, nil
}

func indexHandler(req events.Request) (events.Response, error) {
	// TODO: use nicer homepage template
	// TODO: Show if you're already auth'd
	// TODO: Show link to auth page
	return events.Succeed("Hello!")
}

func missingHandler(_ events.Request) (events.Response, error) {
	return events.Respond(404, "Resource does not exist")
}

func defaultHandler(req events.Request) (events.Response, error) {
	return events.Redirect("https://"+req.Headers["Host"], 303)
}

func main() {
	var err error

	config, err = loadConfig()
	if err != nil {
		panic(err)
	}

	sm = &sessionManager{
		Name: "session",
		Codec: securecookie.New(
			config.SignKey,
			config.EncKey,
		),
	}

	oauthCfg = &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Endpoint:     githubOauth.Endpoint,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
	}

	r := router.Router{
		Routes: []router.Route{
			router.Route{Path: authRegex, Handler: authHandler},
			router.Route{Path: callbackRegex, Handler: callbackHandler},
			router.Route{Path: indexRegex, Handler: indexHandler},
			router.Route{Path: faviconRegex, Handler: missingHandler},
			router.Route{Path: defaultRegex, Handler: defaultHandler},
		},
	}
	r.Start()
}

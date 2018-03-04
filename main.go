package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/akerl/go-lambda/apigw/events"
	"github.com/akerl/go-lambda/apigw/router"
	"github.com/akerl/go-lambda/s3"
	"github.com/google/go-github/github"
	"github.com/gorilla/securecookie"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

type configFile struct {
	ClientSecret string `json:"clientsecret"`
	ClientID     string `json:"clientid"`
	SignKey      string `json:"signkey"`
	EncKey       string `json:"enckey"`
}

const (
	githubAuthorizeURL = "https://github.com/login/oauth/authorize"
	githubTokenURL     = "https://github.com/login/oauth/access_token"
	redirectURL        = ""
)

var (
	config        *configFile
	cookieManager *securecookie.SecureCookie
	oauthCfg      *oauth2.Config
	scopes        = []string{"read:org"}

	authRegex     = regexp.MustCompile(`^/auth$`)
	callbackRegex = regexp.MustCompile(`^/callback$`)
	indexRegex    = regexp.MustCompile(`^/$`)
	defaultRegex  = regexp.MustCompile(`/.*`)
)

func authHandler(req events.Request) (events.Response, error) {
	b := make([]byte, 16)
	rand.Read(b)

	state := base64.URLEncoding.EncodeToString(b)

	session, _ := store.Get(hr, "sess")
	session.Values["state"] = state
	session.Save(r, w)

	url := oauthCfg.AuthCodeURL(state)
	http.Redirect(w, r, url, 302)
	return events.Succeed("Auth!")
}

func callbackHandler(req events.Request) (events.Response, error) {
	session, err := store.Get(r, "sess")
	if err != nil {
		fmt.Fprintln(w, "aborted")
		return
	}

	if r.URL.Query().Get("state") != session.Values["state"] {
		fmt.Fprintln(w, "no state match; possible csrf OR cookies not enabled")
		return
	}

	tkn, err := oauthCfg.Exchange(oauth2.NoContext, r.URL.Query().Get("code"))
	if err != nil {
		fmt.Fprintln(w, "there was an issue getting your token")
		return
	}

	if !tkn.Valid() {
		fmt.Fprintln(w, "retreived invalid token")
		return
	}

	client := github.NewClient(oauthCfg.Client(oauth2.NoContext, tkn))

	user, _, err := client.Users.Get("")
	if err != nil {
		fmt.Println(w, "error getting name")
		return
	}

	session.Values["name"] = user.Name
	session.Values["accessToken"] = tkn.AccessToken
	session.Save(r, w)

	http.Redirect(w, r, "/", 302)

	return events.Succeed("Callback!")
}

func indexHandler(req events.Request) (events.Response, error) {
	// TODO: use nicer homepage template
	// TODO: Show if you're already auth'd
	// TODO: Show link to auth page
	//return events.Succeed("Index!")
	return events.Succeed(fmt.Sprintf("%+v\n", req))
}

func defaultHandler(req events.Request) (events.Response, error) {
	return events.Response{
		StatusCode: 303,
		Headers: map[string]string{
			"Location": "https://" + req.Headers["Host"],
		},
	}, nil
}

func loadConfig() {
	bucket := os.Getenv("S3_BUCKET")
	path := os.Getenv("S3_KEY")
	if bucket == "" || path == "" {
		log.Print("variables not provided")
		return
	}

	obj, err := s3.GetObject(bucket, path)
	if err != nil {
		log.Print(err)
		return
	}

	c := configFile{}
	err = yaml.Unmarshal(obj, &c)
	if err != nil {
		log.Print(err)
		return
	}
	config = &c
}

func main() {
	loadConfig()

	cookieManager = securecookie.New(
		[]byte(config.SignKey),
		[]byte(config.EncKey),
	)

	oauthCfg = &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  githubAuthorizeURL,
			TokenURL: githubTokenURL,
		},
		RedirectURL: redirectURL,
		Scopes:      scopes,
	}

	r := router.Router{
		Routes: []router.Route{
			router.Route{Path: authRegex, Handler: authHandler},
			router.Route{Path: callbackRegex, Handler: callbackHandler},
			router.Route{Path: indexRegex, Handler: indexHandler},
			router.Route{Path: defaultRegex, Handler: defaultHandler},
		},
	}
	r.Start()
}

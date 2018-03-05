package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
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

// TODO: Clean up error messages / logging
// TODO: Return useful HTTP error codes for failure

type configFile struct {
	ClientSecret string `json:"clientsecret"`
	ClientID     string `json:"clientid"`
	SignKey      string `json:"signkey"`
	EncKey       string `json:"enckey"`
}

type sessionManager struct {
	Name  string
	Codec securecookie.Codec
}

type session struct {
	Nonce string   `json:"state"`
	Token string   `json:"token"`
	Login string   `json:"login"`
	Orgs  []string `json:"orgs"`
}

func (sc *sessionManager) Read(req events.Request) (session, error) {
	sess := session{}

	header := http.Header{}
	header.Add("Cookie", req.Headers["Cookie"])
	request := http.Request{Header: header}
	cookie, err := request.Cookie(sm.Name)
	if err == http.ErrNoCookie {
		return sess, nil
	} else if err != nil {
		return sess, fmt.Errorf("failed to read cookie")
	}

	err = sc.Codec.Decode(sm.Name, cookie.Value, &sess)
	if err != nil {
		log.Print("failed to decode cookie")
	}
	return sess, err
}

func (sc *sessionManager) Write(sess session) (string, error) {
	encoded, err := sc.Codec.Encode(sm.Name, sess)
	if err != nil {
		return "", err
	}

	// TODO: Set expiration time / max age
	// TODO: Set domain field
	cookie := &http.Cookie{
		Name:     sm.Name,
		Value:    encoded,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
	}
	return cookie.String(), nil
}

const (
	githubAuthorizeURL = "https://github.com/login/oauth/authorize"
	githubTokenURL     = "https://github.com/login/oauth/access_token"
	redirectURL        = ""
)

var (
	config   *configFile
	sm       *sessionManager
	oauthCfg *oauth2.Config
	scopes   = []string{"read:org"}

	authRegex     = regexp.MustCompile(`^/auth$`)
	callbackRegex = regexp.MustCompile(`^/callback$`)
	indexRegex    = regexp.MustCompile(`^/$`)
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
	errorRedirect, _ := events.Redirect("https://"+req.Headers["Host"], 303)

	sess, err := sm.Read(req)
	if err != nil {
		log.Printf("Failed loading session cookie: %s", err)
		return events.Fail("error; aborting")
	}

	actual := req.QueryStringParameters["state"]

	if sess.Nonce == "" {
		log.Print("callback hit with no nonce")
		return errorRedirect, nil
	} else if sess.Nonce != actual {
		log.Print("nonce mismatch; possible csrf OR cookies not enabled")
		return errorRedirect, nil
	}

	code := req.QueryStringParameters["code"]
	token, err := oauthCfg.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Print("there was an issue getting your token")
		return errorRedirect, nil
	}

	if !token.Valid() {
		log.Print("retreived invalid token")
		return errorRedirect, nil
	}

	client := github.NewClient(oauthCfg.Client(oauth2.NoContext, token))
	user, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		log.Print("error getting name")
		return errorRedirect, nil
	}
	orgs, _, err := client.Organizations.List(context.Background(), "", &github.ListOptions{})
	if err != nil {
		log.Print("error getting orgs")
		return errorRedirect, nil
	}
	var orgList []string
	for _, i := range orgs {
		orgList = append(orgList, *i.Login)
	}

	sess.Token = token.AccessToken
	sess.Login = *user.Name
	sess.Orgs = orgList

	cookie, err := sm.Write(sess)
	if err != nil {
		log.Print("error encoding cookie")
		return errorRedirect, nil
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
	//return events.Succeed("Index!")
	return events.Succeed(fmt.Sprintf("%+v\n", req))
}

func defaultHandler(req events.Request) (events.Response, error) {
	return events.Redirect("https://"+req.Headers["Host"], 303)
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

	sm = &sessionManager{
		Name: "session",
		Codec: securecookie.New(
			[]byte(config.SignKey),
			[]byte(config.EncKey),
		),
	}

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

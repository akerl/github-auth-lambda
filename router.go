//go:generate resources -output static.go -fmt -trim ./assets/ ./assets/*
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/akerl/go-lambda/apigw/events"
	"github.com/google/go-github/github"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

func fail(msg string) (events.Response, error) {
	var id string
	u, err := uuid.NewRandom()
	if err == nil {
		id = u.String()
	} else {
		id = "uuid_gen_failed"
	}
	log.Printf("%s %s", id, msg)
	userError := fmt.Sprintf("server error while processing request: %s", id)
	return events.Fail(userError)
}

func success(req events.Request, sess session) (events.Response, error) {
	target := sess.Target
	sess.Target = ""
	return redirect(req, sess, target)
}

func redirect(req events.Request, sess session, target string) (events.Response, error) {
	if target == "" {
		target = "https://" + req.Headers["Host"]
	}

	cookie, err := sm.Write(sess)
	if err != nil {
		return fail(fmt.Sprintf("error encoding cookie: %s", err))
	}

	return events.Response{
		StatusCode: 303,
		Headers: map[string]string{
			"Location":   target,
			"Set-Cookie": cookie,
		},
	}, nil
}

func defaultHandler(req events.Request) (events.Response, error) {
	return events.Redirect("https://"+req.Headers["Host"], 303)
}

func indexHandler(req events.Request) (events.Response, error) {
	// TODO: use nicer homepage template
	// TODO: Show if you're already auth'd
	// TODO: Show link to auth page
	return events.Succeed("Hello!")
}

func faviconHandler(req events.Request) (events.Response, error) {
	favicon, err := static.ReadFile("favicon.ico")
	if err != nil {
		return missingHandler(req)
	}
	return events.Succeed(string(favicon))
}

func missingHandler(_ events.Request) (events.Response, error) {
	return events.Respond(404, "resource does not exist")
}

func authHandler(req events.Request) (events.Response, error) {
	sess, err := sm.Read(req)
	if err != nil {
		return fail(fmt.Sprintf("failed loading session cookie: %s", err))
	}

	if sess.Login != "" {
		return success(req, sess)
	}

	sess.Target = req.QueryStringParameters["redirect"]

	err = sess.SetNonce()
	if err != nil {
		return fail(fmt.Sprintf("failed to generate nonce: %s", err))
	}

	url := oauthCfg.AuthCodeURL(sess.Nonce)

	return redirect(req, sess, url)
}

func callbackHandler(req events.Request) (events.Response, error) {
	sess, err := sm.Read(req)
	if err != nil {
		return fail(fmt.Sprintf("failed loading session cookie: %s", err))
	}

	if sess.Login != "" {
		return success(req, sess)
	}

	actual := req.QueryStringParameters["state"]

	if sess.Nonce == "" {
		log.Print("callback hit with no nonce")
		return events.Redirect("https://"+req.Headers["Host"], 303)
	} else if sess.Nonce != actual {
		return fail("nonce mismatch")
	}

	code := req.QueryStringParameters["code"]
	token, err := oauthCfg.Exchange(oauth2.NoContext, code)
	if err != nil {
		return fail(fmt.Sprintf("there was an issue getting your token: %s", err))
	}

	if !token.Valid() {
		return fail("retreived invalid token")
	}

	client := github.NewClient(oauthCfg.Client(oauth2.NoContext, token))
	user, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		return fail(fmt.Sprintf("error getting name: %s", err))
	}
	orgs, _, err := client.Organizations.List(context.Background(), "", &github.ListOptions{})

	if err != nil {
		return fail(fmt.Sprintf("error getting orgs: %s", err))
	}
	var orgList []string
	for _, i := range orgs {
		orgList = append(orgList, *i.Login)
	}

	sess.Login = *user.Name
	sess.Orgs = orgList

	return success(req, sess)
}

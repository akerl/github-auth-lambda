//go:generate resources -output static.go -declare -var static -fmt -trim assets/ assets/*
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/akerl/github-auth-lambda/session"

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

func success(req events.Request, sess session.Session) (events.Response, error) {
	target := sess.Target
	sess.Target = ""
	return redirect(req, sess, target)
}

func redirect(req events.Request, sess session.Session, target string) (events.Response, error) {
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
	page, err := execTemplate("/index.html", req)
	if err != nil {
		return fail(fmt.Sprintf("failed to exec template: %s", err))
	}
	return events.Response{
		StatusCode: 200,
		Body:       page,
		Headers: map[string]string{
			"Content-Type": "text/html; charset=utf-8",
		},
	}, nil
}

func faviconHandler(req events.Request) (events.Response, error) {
	favicon, found := static.files["/favicon.ico"]
	if !found {
		return missingHandler(req)
	}

	encodedFavicon := base64.StdEncoding.EncodeToString(favicon.data)
	return events.Response{
		StatusCode: 200,
		Body:       encodedFavicon,
		Headers: map[string]string{
			"Content-Type": "image/x-icon",
		},
		IsBase64Encoded: true,
	}, nil
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

	if sess.Target == "" {
		sess.Target = req.QueryStringParameters["redirect"]
	}

	err = sess.SetNonce()
	if err != nil {
		return fail(fmt.Sprintf("failed to generate nonce: %s", err))
	}

	url := oauthCfg.AuthCodeURL(sess.Nonce)

	return redirect(req, sess, url)
}

func logoutHandler(req events.Request) (events.Response, error) {
	return redirect(req, session.Session{}, "")
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
	sess.Login = *user.Login

	teams, _, err := client.Organizations.ListUserTeams(context.Background(), &github.ListOptions{})
	if err != nil {
		return fail(fmt.Sprintf("error getting teams: %s", err))
	}
	sess.Memberships = make(map[string][]string)
	for _, t := range teams {
		org := *t.Organization.Login
		sess.Memberships[org] = append(sess.Memberships[org], *t.Slug)
	}

	return success(req, sess)
}

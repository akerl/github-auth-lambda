package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/akerl/go-lambda/apigw/events"
	"github.com/gorilla/securecookie"
)

type session struct {
	Nonce string   `json:"state"`
	Login string   `json:"login"`
	Orgs  []string `json:"orgs"`
}

type sessionManager struct {
	Name  string
	Codec securecookie.Codec
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
		return sess, fmt.Errorf("failed to read cookie: %s", err)
	}

	err = sc.Codec.Decode(sm.Name, cookie.Value, &sess)
	if err != nil {
		log.Printf("failed to decode cookie: %s", err)
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

package main

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/akerl/go-lambda/apigw/events"
	"github.com/gorilla/securecookie"
)

type session struct {
	Nonce  string   `json:"state"`
	Login  string   `json:"login"`
	Orgs   []string `json:"orgs"`
	Target string   `json:"target"`
}

func (s *session) SetNonce() error {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return err
	}
	s.Nonce = base64.URLEncoding.EncodeToString(b)
	return nil
}

type sessionManager struct {
	Name     string
	SignKey  []byte
	EncKey   []byte
	Lifetime int
	Domain   string
	codec    *securecookie.SecureCookie
}

func (sc *sessionManager) initCodec() {
	if sc.codec != nil {
		return
	}
	sc.codec = securecookie.New(
		sc.SignKey,
		sc.EncKey,
	)
	sc.codec.MaxAge(sc.Lifetime)
}

func (sc *sessionManager) decode(name, cookie string, sess *session) error {
	sc.initCodec()
	return sc.codec.Decode(name, cookie, &sess)
}

func (sc *sessionManager) encode(name string, sess session) (string, error) {
	sc.initCodec()
	return sc.codec.Encode(name, sess)
}

func (sc *sessionManager) Read(req events.Request) (session, error) {
	header := http.Header{}
	header.Add("Cookie", req.Headers["Cookie"])
	request := http.Request{Header: header}
	cookie, err := request.Cookie(sm.Name)
	if err == http.ErrNoCookie {
		return session{}, nil
	} else if err != nil {
		return session{}, err
	}

	s := session{}
	err = sc.decode(sm.Name, cookie.Value, &s)
	if err == nil {
		return s, nil
	}
	if scError, ok := err.(securecookie.Error); ok && scError.IsDecode() {
		return session{}, nil
	}
	return session{}, err
}

func (sc *sessionManager) Write(sess session) (string, error) {
	encoded, err := sc.encode(sm.Name, sess)
	if err != nil {
		return "", err
	}

	cookie := &http.Cookie{
		Name:     sm.Name,
		Value:    encoded,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		MaxAge:   sc.Lifetime,
		Domain:   sc.Domain,
	}

	return cookie.String(), nil
}

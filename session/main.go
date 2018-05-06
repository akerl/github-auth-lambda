package session

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/akerl/go-lambda/apigw/events"
	"github.com/gorilla/securecookie"
)

// Session demribes the Session object
type Session struct {
	Nonce       string              `json:"state"`
	Login       string              `json:"login"`
	Memberships map[string][]string `json:"memberships"`
	Target      string              `json:"target"`
}

// SetNonce sets the nonce for the Session object
func (s *Session) SetNonce() error {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return err
	}
	s.Nonce = base64.URLEncoding.EncodeToString(b)
	return nil
}

// Manager handles encoding/decoding cookies
type Manager struct {
	Name     string
	SignKey  []byte
	EncKey   []byte
	Lifetime int
	Domain   string
	codec    *securecookie.SecureCookie
}

func (m *Manager) initCodec() {
	if m.codec != nil {
		return
	}
	m.codec = securecookie.New(
		m.SignKey,
		m.EncKey,
	)
	m.codec.MaxAge(m.Lifetime)
}

func (m *Manager) decode(name, cookie string, sess *Session) error {
	m.initCodec()
	return m.codec.Decode(name, cookie, &sess)
}

func (m *Manager) encode(name string, sess Session) (string, error) {
	m.initCodec()
	return m.codec.Encode(name, sess)
}

// Read reads a cookie from a request
func (m *Manager) Read(req events.Request) (Session, error) {
	header := http.Header{}
	header.Add("Cookie", req.Headers["Cookie"])
	request := http.Request{Header: header}
	cookie, err := request.Cookie(m.Name)
	if err == http.ErrNoCookie {
		return Session{}, nil
	} else if err != nil {
		return Session{}, err
	}

	s := Session{}
	err = m.decode(m.Name, cookie.Value, &s)
	if err == nil {
		return s, nil
	}
	if mError, ok := err.(securecookie.Error); ok && mError.IsDecode() {
		return Session{}, nil
	}
	return Session{}, err
}

// Write encodes a cookie from a Session
func (m *Manager) Write(sess Session) (string, error) {
	encoded, err := m.encode(m.Name, sess)
	if err != nil {
		return "", err
	}

	cookie := &http.Cookie{
		Name:     m.Name,
		Value:    encoded,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		MaxAge:   m.Lifetime,
		Domain:   m.Domain,
	}

	return cookie.String(), nil
}

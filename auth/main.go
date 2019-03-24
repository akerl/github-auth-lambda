package auth

import (
	"net/url"

	"github.com/akerl/github-auth-lambda/session"

	"github.com/akerl/go-lambda/apigw/events"
)

// SessionCheck defines a helper for checking session validity
type SessionCheck struct {
	SessionManager session.Manager
	AuthURL        string
	ACLHandler     func(events.Request, session.Session) (bool, error)
}

// AuthFunc checks for valid auth using GitHub OAuth
func (sc *SessionCheck) AuthFunc(req events.Request) (events.Response, error) {
	sess, err := sc.SessionManager.Read(req)
	if err != nil {
		return events.Fail("failed to authenticate request")
	}

	if sess.Login == "" {
		authURL, err := url.Parse(sc.AuthURL)
		if err != nil {
			return events.Response{}, err
		}

		returnURL := url.URL{
			Host:   req.Headers["Host"],
			Path:   req.Path,
			Scheme: "https",
		}
		returnValues := authURL.Query()
		returnValues.Set("redirect", returnURL.String())
		authURL.RawQuery = returnValues.Encode()

		return events.Redirect(authURL.String(), 303)
	}

	allowed, err := sc.ACLHandler(req, sess)
	if err != nil {
		return events.Fail("failed to authenticate request")
	}
	if allowed {
		return events.Response{}, nil
	}
	return events.Reject("Not authorized")
}

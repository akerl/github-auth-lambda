package main

import (
	"encoding/base64"
	"fmt"

	"github.com/akerl/go-lambda/s3"
)

type configFile struct {
	ClientSecret  string            `json:"clientsecret"`
	ClientID      string            `json:"clientid"`
	Lifetime      int               `json:"lifetime"`
	Domain        string            `json:"domain"`
	Base64SignKey string            `json:"signkey"`
	Base64EncKey  string            `json:"enckey"`
	SignKey       []byte            `json:"-"`
	EncKey        []byte            `json:"-"`
	TemplateData  map[string]string `json:"templatedata"`
}

func loadConfig() (*configFile, error) {
	c := configFile{}
	cf, err := s3.GetConfigFromEnv(&c)
	if err != nil {
		return &c, err
	}
	cf.OnError = func(_ *s3.ConfigFile, err error) {
		fmt.Println(err)
	}
	cf.Autoreload(60)

	if c.Lifetime == 0 {
		c.Lifetime = 86400
	}

	if c.ClientSecret == "" || c.ClientID == "" {
		return &c, fmt.Errorf("clientid and clientsecret not set")
	}

	if c.Base64SignKey == "" || c.Base64EncKey == "" {
		return &c, fmt.Errorf("signing and encryption keys not set")
	}

	c.SignKey, err = base64.URLEncoding.DecodeString(c.Base64SignKey)
	if err != nil {
		return &c, err
	}
	c.EncKey, err = base64.URLEncoding.DecodeString(c.Base64EncKey)
	if err != nil {
		return &c, err
	}

	return &c, nil
}

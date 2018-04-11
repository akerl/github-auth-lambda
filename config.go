package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/akerl/go-lambda/s3"
	"gopkg.in/yaml.v2"
)

type configFile struct {
	ClientSecret  string `json:"clientsecret"`
	ClientID      string `json:"clientid"`
	Base64SignKey string `json:"signkey"`
	Base64EncKey  string `json:"enckey"`
	SignKey       []byte `json:"-"`
	EncKey        []byte `json:"-"`
}

func loadConfig() (*configFile, error) {
	c := configFile{}

	bucket := os.Getenv("S3_BUCKET")
	path := os.Getenv("S3_KEY")
	if bucket == "" || path == "" {
		return &c, fmt.Errorf("variables not provided")
	}

	obj, err := s3.GetObject(bucket, path)
	if err != nil {
		return &c, err
	}

	err = yaml.Unmarshal(obj, &c)
	if err != nil {
		return &c, err
	}

	if c.ClientSecret == "" || c.ClientID == "" {
		return &c, fmt.Errorf("Client ID and Secret must be set")
	}

	if c.Base64SignKey == "" || c.Base64EncKey == "" {
		return &c, fmt.Errorf("Signing and encryption keys not set")
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

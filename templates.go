package main

import (
	"fmt"

	"github.com/akerl/go-lambda/apigw/events"
	"github.com/aymerick/raymond"
)

var (
	templateNames = []string{
		"/index.html",
	}
	templates = map[string]*raymond.Template{}
)

func loadTemplate(name string) error {
	var err error

	tplName := fmt.Sprintf("%s.hbs", name)
	tplFile, found := static.String(tplName)
	if !found {
		return fmt.Errorf("template not found: %s", tplFile)
	}
	templates[name], err = raymond.Parse(tplFile)
	if err != nil {
		return fmt.Errorf("template failed to parse (%s): %s", tplFile, err)
	}
	return nil
}

func loadTemplates() error {
	for _, name := range templateNames {
		err := loadTemplate(name)
		if err != nil {
			return err
		}
	}
	return nil
}

func init() {
	err := loadTemplates()
	if err != nil {
		panic(err)
	}
}

type templateContext struct {
	Session session
	Request events.Request
	Config  map[string]string
}

func newTemplateContext(req events.Request) (templateContext, error) {
	tc := templateContext{
		Request: req,
		Config:  config.TemplateData,
	}

	var err error
	tc.Session, err = sm.Read(req)
	return tc, err
}

func execTemplate(name string, req events.Request) (string, error) {
	ctx, err := newTemplateContext(req)
	if err != nil {
		return "", err
	}

	tpl, found := templates[name]
	if !found {
		return "", fmt.Errorf("template does not exist: %s", name)
	}

	page, err := tpl.Exec(ctx)
	return page, err
}
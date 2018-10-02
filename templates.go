package main

import (
	"fmt"

	"github.com/akerl/go-lambda/apigw/events"
	"gopkg.in/osteele/liquid.v1"
)

var (
	engine        *liquid.Engine
	templateNames = []string{
		"/index.html",
	}
	templates = map[string]*liquid.Template{}
)

func loadTemplate(name string) error {
	var err error

	tplName := fmt.Sprintf("%s.hbs", name)
	tplFile, found := static.String(tplName)
	if !found {
		return fmt.Errorf("template not found: %s", tplFile)
	}
	templates[name], err = engine.ParseString(tplFile)
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
	engine = liquid.NewEngine()
	err := loadTemplates()
	if err != nil {
		panic(err)
	}
}

func newTemplateContext(req events.Request) (map[string]interface{}, error) {
	session, err := sm.Read(req)
	if err != nil {
		return map[string]interface{}{}, err
	}

	orgs := make([]string, len(tcsession.Memberships))
	for idx, org := range session.Memberships {
		orgs[idx] = org
	}
	sort.Strings(orgs)

	tc := map[string]interface{}{
		"request": req,
		"config":  config.TemplateData,
		"session": session,
		"orgs":    orgs,
	}
	return tc, nil
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

	page, err := tpl.RenderString(ctx)
	return page, err
}

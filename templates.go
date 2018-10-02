package main

import (
	"fmt"
	"sort"

	"github.com/akerl/github-auth-lambda/session"

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
	templates[name].RegisterHelper("each_team", eachTeamHelper)
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
	Session session.Session
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

func eachTeamHelper(memberships map[string][]string, options *raymond.Options) string {
	result := ""

	orgCount := len(memberships)
	orgs := make([]string, orgCount)
	idx := 0
	for key := range memberships {
		orgs[idx] = key
		idx++
	}
	sort.Strings(orgs)

	for idx, key := range orgs {
		val := memberships[key]
		sort.Strings(val)
		data := options.newIterDataFrame(orgCount, idx, key)
		result += options.evalBlock(val, data, key)
	}

	return result
}

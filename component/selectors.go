package component

import (
	"context"
	"fmt"

	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/component/models"
	"github.com/anchordotdev/cli/ui"
)

type OrgChoices []api.Organization

func (OrgChoices) Flag() string     { return "--org" }
func (OrgChoices) Plural() string   { return "organizations" }
func (OrgChoices) Singular() string { return "organization" }

func (c OrgChoices) ListItems() []ui.ListItem[api.Organization] {
	var items []ui.ListItem[api.Organization]
	for _, org := range c {
		item := ui.ListItem[api.Organization]{
			Key:    org.Apid,
			String: fmt.Sprintf("%s (%s)", org.Name, org.Apid),
			Value:  org,
		}
		items = append(items, item)
	}
	return items
}

func OrgSelector(ctx context.Context, drv *ui.Driver, anc *api.Session, prompt string) (chan api.Organization, error) {
	choicec := make(chan api.Organization, 1)

	drv.Activate(ctx, &models.SelectorFetcher[api.Organization, OrgChoices]{})
	orgs, err := anc.GetOrgs(ctx)
	if err != nil {
		return nil, err
	}
	choices := OrgChoices(orgs)
	drv.Send(choices)

	if len(orgs) == 1 {
		choicec <- orgs[0]
		return choicec, nil
	}

	drv.Activate(ctx, &models.Selector[api.Organization]{
		ChoiceCh: choicec,
		Choices:  choices,
		Prompt:   prompt,
	})

	return choicec, nil
}

type RealmChoices []api.Realm

func (RealmChoices) Flag() string     { return "--realm" }
func (RealmChoices) Plural() string   { return "realms" }
func (RealmChoices) Singular() string { return "realm" }

func (c RealmChoices) ListItems() []ui.ListItem[api.Realm] {
	var items []ui.ListItem[api.Realm]
	for _, realm := range c {
		item := ui.ListItem[api.Realm]{
			Key:    realm.Apid,
			String: fmt.Sprintf("%s (%s)", realm.Name, realm.Apid),
			Value:  realm,
		}
		items = append(items, item)
	}
	return items
}

func RealmSelector(ctx context.Context, drv *ui.Driver, anc *api.Session, orgApid string, prompt string) (chan api.Realm, error) {
	choicec := make(chan api.Realm, 1)

	drv.Activate(ctx, &models.SelectorFetcher[api.Realm, RealmChoices]{})

	realms, err := anc.GetOrgRealms(ctx, orgApid)
	if err != nil {
		return nil, err
	}

	choices := RealmChoices(realms)
	drv.Send(choices)

	if len(realms) == 1 {
		choicec <- realms[0]
		return choicec, nil
	}

	drv.Activate(ctx, &models.Selector[api.Realm]{
		ChoiceCh: choicec,
		Choices:  choices,
		Prompt:   prompt,
	})

	return choicec, nil
}

type ServiceChoices []api.Service

func (ServiceChoices) Flag() string     { return "--service" }
func (ServiceChoices) Plural() string   { return "services" }
func (ServiceChoices) Singular() string { return "service" }

func (c ServiceChoices) ListItems() []ui.ListItem[api.Service] {
	var items []ui.ListItem[api.Service]
	for _, service := range c {
		item := ui.ListItem[api.Service]{
			Key:    service.Slug,
			String: fmt.Sprintf("%s (%s)", service.Name, service.Slug),
			Value:  service,
		}
		items = append(items, item)
	}
	return items
}

func ServiceSelector(ctx context.Context, drv *ui.Driver, anc *api.Session, orgApid string, prompt string) (chan api.Service, error) {
	choicec := make(chan api.Service, 1)

	drv.Activate(ctx, &models.SelectorFetcher[api.Service, ServiceChoices]{})

	services, err := anc.GetOrgServices(ctx, orgApid)
	if err != nil {
		return nil, err
	}

	choices := ServiceChoices(services)
	drv.Send(choices)

	if len(services) == 1 {
		choicec <- services[0]
		return choicec, nil
	}

	drv.Activate(ctx, &models.Selector[api.Service]{
		ChoiceCh: choicec,
		Choices:  choices,
		Prompt:   prompt,
	})

	return choicec, nil
}

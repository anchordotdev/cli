package lcl

import (
	"context"
	"crypto/tls"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/lcl/models"
	"github.com/anchordotdev/cli/ui"
)

type Provision struct {
	Config *cli.Config

	Domains            []string
	orgSlug, realmSlug string
}

func (p *Provision) run(ctx context.Context, drv *ui.Driver, anc *api.Session, serviceName, serverType string) (*api.Service, *tls.Certificate, error) {
	drv.Activate(ctx, &models.ProvisionService{
		Name:       serviceName,
		Domains:    p.Domains,
		ServerType: serverType,
	})

	userInfo, err := anc.UserInfo(ctx)
	if err != nil {
		return nil, nil, err
	}

	if p.orgSlug == "" {
		p.orgSlug = userInfo.PersonalOrg.Slug
	}
	serviceParam := serviceName

	srv, err := anc.GetService(ctx, p.orgSlug, serviceParam)
	if err != nil {
		return nil, nil, err
	}
	if srv == nil {
		srv, err = anc.CreateService(ctx, p.orgSlug, serverType, serviceParam)
		if err != nil {
			return nil, nil, err
		}
	}

	// FIXME: we need to lookup and pass the chain and/or make it non-optional
	chainParam := "ca"

	attach, err := anc.AttachService(ctx, chainParam, p.Domains, p.orgSlug, p.realmSlug, serviceParam)
	if err != nil {
		return nil, nil, err
	}

	cmdMkCert := &MkCert{
		Config:      p.Config,
		anc:         anc,
		chainSlug:   attach.Relationships.Chain.Slug,
		domains:     p.Domains,
		orgSlug:     p.orgSlug,
		realmSlug:   p.realmSlug,
		serviceSlug: serviceParam,
		subCaSubjectUID:  attach.Relationships.SubCa.Slug,
	}

	tlsCert, err := cmdMkCert.perform(ctx, drv)
	if err != nil {
		return nil, nil, err
	}

	drv.Send(models.ServiceProvisionedMsg{})

	return srv, tlsCert, nil
}

package api

import "slices"

func (o Organization) Key() string      { return o.Apid }
func (o Organization) String() string   { return o.Name }
func (o Organization) Plural() string   { return "organizations" }
func (o Organization) Singular() string { return "organization" }

func (r Realm) Key() string      { return r.Apid }
func (r Realm) String() string   { return r.Name }
func (r Realm) Plural() string   { return "realms" }
func (r Realm) Singular() string { return "realm" }

func (s Service) Key() string      { return s.Slug }
func (s Service) String() string   { return s.Name }
func (s Service) Plural() string   { return "services" }
func (s Service) Singular() string { return "service" }

func NonDiagnosticServices(s []Service) []Service {
	return slices.DeleteFunc(s, func(svc Service) bool {
		return svc.ServerType == ServiceServerTypeDiagnostic
	})
}

var _ Filter[Service] = NonDiagnosticServices

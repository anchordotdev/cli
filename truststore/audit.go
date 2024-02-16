package truststore

import (
	"slices"
	"time"
)

type Audit struct {
	Expected []*CA

	Stores []Store

	At time.Time

	SelectFn func(*CA) (bool, error)
}

type AuditInfo struct {
	Valid, Missing, Rotate, Expired, PreValid, Extra []*CA

	casByStore map[Store]map[string]*CA
}

func (i *AuditInfo) AllCAs(states ...string) []*CA {
	var cas []*CA
	if slices.Contains(states, "valid") || slices.Contains(states, "all") {
		cas = append(cas, i.Valid...)
	}
	if slices.Contains(states, "rotate") || slices.Contains(states, "all") {
		cas = append(cas, i.Rotate...)
	}
	if slices.Contains(states, "expired") || slices.Contains(states, "all") {
		cas = append(cas, i.Expired...)
	}
	if slices.Contains(states, "prevalid") || slices.Contains(states, "all") {
		cas = append(cas, i.PreValid...)
	}
	if slices.Contains(states, "extra") || slices.Contains(states, "all") {
		cas = append(cas, i.Extra...)
	}
	return cas
}

func (i *AuditInfo) IsPresent(ca *CA, store Store) bool {
	_, ok := i.casByStore[store][ca.UniqueName]
	return ok
}

func (a *Audit) Perform() (*AuditInfo, error) {
	if a.At.IsZero() {
		a.At = time.Now()
	}

	info := &AuditInfo{
		casByStore: make(map[Store]map[string]*CA),
	}

	casByName := make(map[string]*CA)
	storesByCA := make(map[string][]Store)
	for _, store := range a.Stores {
		cas, err := store.ListCAs()
		if err != nil {
			return nil, err
		}

		for _, ca := range cas {
			if a.SelectFn != nil {
				if keep, err := a.SelectFn(ca); err != nil {
					return nil, err
				} else if !keep {
					continue
				}
			}

			casByName[ca.UniqueName] = ca
			storesByCA[ca.UniqueName] = append(storesByCA[ca.UniqueName], store)

			set, ok := info.casByStore[store]
			if !ok {
				set = make(map[string]*CA)
			}
			set[ca.UniqueName] = ca

			info.casByStore[store] = set
		}
	}

	partialValid := make(map[string]*CA)
	for _, ca := range a.Expected {
		if _, ok := casByName[ca.UniqueName]; ok {
			switch {
			case a.isExpired(ca):
				info.Expired = append(info.Expired, ca)
			case a.isPreValid(ca):
				info.Expired = append(info.PreValid, ca)
			case a.isRotate(ca):
				info.Rotate = append(info.Rotate, ca)
			default:
				partialValid[ca.UniqueName] = ca
			}

			delete(casByName, ca.UniqueName)
		} else {
			info.Missing = append(info.Missing, ca)
		}
	}

	for _, ca := range casByName {
		info.Extra = append(info.Extra, ca)

		for _, store := range storesByCA[ca.UniqueName] {
			set, ok := info.casByStore[store]
			if !ok {
				set = make(map[string]*CA)
			}
			set[ca.UniqueName] = ca

			info.casByStore[store] = set
		}
	}

	for _, ca := range partialValid {
		if len(storesByCA[ca.UniqueName]) < len(a.Stores) {
			info.Missing = append(info.Missing, ca)
		} else {
			info.Valid = append(info.Valid, ca)
		}
	}
	return info, nil
}

func (a *Audit) isExpired(ca *CA) bool { return a.At.After(ca.NotAfter.Add(1 * time.Second)) }

func (a *Audit) isPreValid(ca *CA) bool { return a.At.Before(ca.NotBefore) }

func (a *Audit) isRotate(ca *CA) bool {
	// TODO: lookup renew value from the extension
	return false
}

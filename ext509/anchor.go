package ext509

import (
	"crypto/x509/pkix"
	"errors"
	"time"

	"golang.org/x/crypto/cryptobyte"
	xasn1 "golang.org/x/crypto/cryptobyte/asn1"

	"github.com/anchordotdev/cli/ext509/oid"
)

// AnchorCertificate is a custom X.509 certificate extension used by Anchor
// Security, Inc. to include important certificate metadata.
type AnchorCertificate struct {
	AutoRenewAt time.Time
	RenewAfter  time.Time
}

func (ac AnchorCertificate) Extension() (pkix.Extension, error) {
	var b cryptobyte.Builder
	b.AddValue(ac)

	buf, err := b.Bytes()
	if err != nil {
		return pkix.Extension{}, err
	}

	return pkix.Extension{
		Id:       oid.AnchorCertificateExtension,
		Critical: false,
		Value:    buf,
	}, nil
}

var (
	tagAutoRenewAt = xasn1.Tag(1).Constructed().ContextSpecific()
	tagRenewAfter  = xasn1.Tag(2).Constructed().ContextSpecific()
)

func (ac AnchorCertificate) Marshal(b *cryptobyte.Builder) error {
	// Anchor ::= SEQUENCE {
	//	_reserved_	[0]	RESERVED	OPTIONAL,
	//	autoRenewAt	[1]	GeneralizedTime	OPTIONAL,
	//	renewAfter	[2]	GeneralizedTime	OPTIONAL }
	b.AddASN1(xasn1.SEQUENCE, func(b *cryptobyte.Builder) {
		if !ac.AutoRenewAt.IsZero() {
			b.AddASN1(tagAutoRenewAt, func(b *cryptobyte.Builder) {
				b.AddASN1GeneralizedTime(ac.AutoRenewAt.UTC().Round(time.Second))
			})
		}

		if !ac.RenewAfter.IsZero() {
			b.AddASN1(tagRenewAfter, func(b *cryptobyte.Builder) {
				b.AddASN1GeneralizedTime(ac.RenewAfter.UTC().Round(time.Second))
			})
		}
	})
	return nil
}

func (ac *AnchorCertificate) Unmarshal(ext pkix.Extension) error {
	if !ext.Id.Equal(oid.AnchorCertificateExtension) || ext.Critical {
		return errors.New("ext509: not an Anchor Certificate Extension")
	}

	input := cryptobyte.String(ext.Value)
	if !input.ReadASN1(&input, xasn1.SEQUENCE) {
		return errors.New("ext509: malformed Anchor Certificate Extension")
	}

	for !input.Empty() {
		var (
			buf cryptobyte.String
			tag xasn1.Tag
		)

		if !input.ReadAnyASN1(&buf, &tag) {
			return errors.New("ext509: malformed Anchor Certificate Extension")
		}

		switch tag {
		case tagAutoRenewAt:
			if !buf.ReadASN1GeneralizedTime(&ac.AutoRenewAt) {
				return errors.New("ext509: malformed Anchor Certificate Extension: autoRenewAt")
			}
		case tagRenewAfter:
			if !buf.ReadASN1GeneralizedTime(&ac.RenewAfter) {
				return errors.New("ext509: malformed Anchor Certificate Extension: renewAfter")
			}
		}
	}
	return nil
}

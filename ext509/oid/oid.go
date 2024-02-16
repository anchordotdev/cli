package oid

import "encoding/asn1"

var (
	AnchorPEN                  = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 60900}
	AnchorCertificateExtension = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 60900, 1}
)

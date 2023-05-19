package must

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"
)

type Certificate tls.Certificate

func CA(template *x509.Certificate) *Certificate {
	template.IsCA = true
	template.BasicConstraintsValid = true

	return Cert(template).SelfSign()
}

func Cert(template *x509.Certificate) *Certificate {
	leaf := new(x509.Certificate)
	*leaf = *template

	sigAlgo := leaf.SignatureAlgorithm
	if sigAlgo == x509.UnknownSignatureAlgorithm {
		sigAlgo = x509.ECDSAWithSHA256
	}

	priv := generateKey(sigAlgo)
	if leaf.PublicKey == nil {
		leaf.PublicKey = priv.Public()
	}

	if leaf.SerialNumber == nil {
		serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
		serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
		if err != nil {
			panic(err)
		}

		leaf.SerialNumber = serialNumber
		leaf.SubjectKeyId = serialNumber.Bytes()
	}

	if leaf.NotAfter.IsZero() {
		now := time.Now()

		leaf.NotBefore = now.Add(-5 * time.Second)
		leaf.NotAfter = now.Add(5 * time.Minute)
	}

	return &Certificate{
		PrivateKey: priv,
		Leaf:       leaf,
	}
}

func (c *Certificate) New(name string) *Certificate {
	template := &x509.Certificate{
		Subject: pkix.Name{
			CommonName: name,
		},
		DNSNames: []string{name},
	}

	return c.Sign(Cert(template))
}
func (c *Certificate) Issue(template *x509.Certificate) *Certificate {
	return c.Sign(Cert(template))
}

func (c *Certificate) SelfSign() *Certificate { return c.Sign(c) }

func (c *Certificate) Sign(template *Certificate) *Certificate {
	cert, err := x509.CreateCertificate(rand.Reader, template.Leaf, c.Leaf, template.Leaf.PublicKey, c.PrivateKey)
	if err != nil {
		panic(err)
	}

	leaf, err := x509.ParseCertificate(cert)
	if err != nil {
		panic(err)
	}

	return &Certificate{
		Certificate: [][]byte{cert},
		PrivateKey:  template.PrivateKey,
		Leaf:        leaf,
	}
}

func (c *Certificate) CertPool() *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AddCert(c.X509())
	return pool
}

func (c *Certificate) TLS() tls.Certificate { return (tls.Certificate)(*c) }

func (c *Certificate) X509() *x509.Certificate { return c.Leaf }

func generateKey(sa x509.SignatureAlgorithm) (key crypto.Signer) {
	var err error
	switch sa {
	case x509.MD2WithRSA, x509.MD5WithRSA, x509.SHA1WithRSA, x509.SHA256WithRSA, x509.SHA384WithRSA, x509.SHA512WithRSA, x509.SHA256WithRSAPSS, x509.SHA384WithRSAPSS, x509.SHA512WithRSAPSS:
		key, err = rsa.GenerateKey(rand.Reader, 2048)
	case x509.ECDSAWithSHA1, x509.ECDSAWithSHA256, x509.ECDSAWithSHA384, x509.ECDSAWithSHA512:
		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case x509.PureEd25519:
		_, key, err = ed25519.GenerateKey(rand.Reader)
	}

	if err != nil {
		panic(err)
	}
	return key
}

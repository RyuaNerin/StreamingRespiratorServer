package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
)

var (
	tlsConfig = tls.Config{
		MinVersion:       tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		CipherSuites: []uint16{
			// generated 2020-04-26, Mozilla Guideline v5.4, Golang 1.14.2, intermediate configuration, no HSTS
			// https://ssl-config.mozilla.org/#server=golang&version=1.14.2&config=intermediate&hsts=false&guideline=5.4
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		},
		PreferServerCipherSuites: false,
		NextProtos:               []string{"http/1.1"},
	}
)

func init() {
	const (
		constCaCert = `
-----BEGIN CERTIFICATE-----
MIIDaTCCAlGgAwIBAgIUVYaTqYZORGZlxyJC+DfGypUy3TcwDQYJKoZIhvcNAQEL
BQAwTjELMAkGA1UEBhMCS1IxHTAbBgNVBAoMFFN0cmVhbWluZyBSZXNwaXJhdG9y
MSAwHgYDVQQDDBdTdHJlYW1pbmcgUmVzcGlyYXRvciBDQTAeFw0yMDAyMDkxMDM1
MzBaFw0zMDAyMDYxMDM1MzBaME4xCzAJBgNVBAYTAktSMR0wGwYDVQQKDBRTdHJl
YW1pbmcgUmVzcGlyYXRvcjEgMB4GA1UEAwwXU3RyZWFtaW5nIFJlc3BpcmF0b3Ig
Q0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC5ntyp7ICGgxOf4QdM
DXG+9T5LfE8wLm1Vkdt8vhqy+81+GZt/TrmDuBcWYdBu3QMKzmQwJVpc9cHU3TLM
EMyXr1fcbsvwRphxrtmUtDKRWYDTu6cDvkYvTZQndBWvD3+MJkbHD6k+fnhWfHbk
KziZeccSg2HT0JSrnSeEyQsaaswMs2xslvx9iSGUbq+OuxB+IEHZ9gCiyWMiH59h
XkJgYkVZcBGfWUebJPFmtiM0ZCvKe9CCABdEblyKfDcrf71ceTPX0OWkxAokyYR+
pblfGSE6SIrZ47HjipRdEYft6CKsFolZC0/RXd69Yv+Dc9zNSYnaOxGOpybHnR0K
XMDBAgMBAAGjPzA9MA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFN16y23Jy0H4
l701uaeqbNNmxm42MAsGA1UdDwQEAwICBDANBgkqhkiG9w0BAQsFAAOCAQEABXkC
RkiQdbNx+a4E/Wkwq45qAj7+Is1Q6mbLTqZZFtSSOeFLWl84K+X55sKwfJ1MjNZF
/Int99WrYuawP+N04hVySXDdNdEwE7FlXRTZXlOIUnH9ixmR2j/w3FLWT3KVXPUk
FceBUTJlfQGSNKM39P+6Tpiiyt0xeCYXObnKVDXLRgkpp5yIVEUGXEradZX/D3gx
P/0oJdVN+4ui/pzz7Df8Wye9+pkjClCmsHRNu4tn0ZmZYQ664UU4U5jtmlrW+qQE
O9Zog7amYye5PKFeD5XSKBWOekzW9/BZ3xzIkpZqFb6pBkllcRdwd/DUZJ2Hvcsu
lzfJN/iCrFvWEttAgg==
-----END CERTIFICATE-----
`

		constClientCert = `
-----BEGIN CERTIFICATE-----
MIID3TCCAsWgAwIBAgIUDudUgRvIMjQgMI6sK5ev1RhTXLQwDQYJKoZIhvcNAQEL
BQAwTjELMAkGA1UEBhMCS1IxHTAbBgNVBAoMFFN0cmVhbWluZyBSZXNwaXJhdG9y
MSAwHgYDVQQDDBdTdHJlYW1pbmcgUmVzcGlyYXRvciBDQTAeFw0yMDAzMTkxNDM1
MzNaFw0zMDAzMTcxNDM1MzNaME4xHTAbBgNVBAoMFFN0cmVhbWluZyBSZXNwaXJh
dG9yMS0wKwYDVQQDDCRTdHJlYW1pbmcgUmVzcGlyYXRvciBQcm94eSBBdXRob3Jp
dHkwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDDEF4nwGh8cvnQ/45l
XzTpgtVz7dMBN/segX8Xb0vvuG1ji8AfrP0apS00dE8QzsjuzN0McShriXPK2kqF
/+iyQeNyTrmtsc2pG6T1I2QhBcC9e06lAvHWwxnDY6t2Sj8xBLJjgCrxuFuWsYsq
gLxlV3HCdgmXTY+JwZgcsxuKRsZVf1yYlr+u1MRrDdvLIDqAmLXtygB+NwqvM+j8
VBl1QDdAxZ5p2EravPk0bey6KlUpYKiT1VV2zl4lzXt2vc2w+20E/vgfPZ3VEPLS
khBt3bUoigyVGEMUaLouPdQXaZd31zFk5WOyBQJQpi0WWD7RhyKrplSr2Srjn9h/
5tgHAgMBAAGjgbIwga8wDAYDVR0TAQH/BAIwADAdBgNVHQ4EFgQUpIUQKdtunRid
YLfZOiHAbng/VjUwHwYDVR0jBBgwFoAU3XrLbcnLQfiXvTW5p6ps02bGbjYwCwYD
VR0PBAQDAgXgMBMGA1UdJQQMMAoGCCsGAQUFBwMBMD0GA1UdEQQ2MDSCFnVzZXJz
dHJlYW0udHdpdHRlci5jb22CD2FwaS50d2l0dGVyLmNvbYIJbG9jYWxob3N0MA0G
CSqGSIb3DQEBCwUAA4IBAQAbEnVE2kQBJAVl38v5xK+WTkO4oYAoLBy1XmRN3VeW
3/ArDajdKrjCqJAi8pUlPgeelBJ5/+0d/hNEvA73s8fvqOnCbFHnOziQ4lHXPwhC
M1C/1ojzUwRBBmz0sOkxtw1hpx6KH5nJDW1MhB6HIlqyHDwskiUFqwgLl4rQGlh2
TdGYAoFZu1qVzHIR1NkcdM+45AFMk7d+0Lo8g3RjrXr9ASXkPfCNVQmDQ0eosOiT
mbn6GQJ1puF7ruIRRj3l5O3dXJYH/nUhl73oC2pWEnnpMAjCycrgTC+nY2jFEzXD
8cxakrMcluwMxUlWKvw+6ZcHkK5xYo/o+inZ1Il+WX/z
-----END CERTIFICATE-----
`

		constClientKey = `
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAwxBeJ8BofHL50P+OZV806YLVc+3TATf7HoF/F29L77htY4vA
H6z9GqUtNHRPEM7I7szdDHEoa4lzytpKhf/oskHjck65rbHNqRuk9SNkIQXAvXtO
pQLx1sMZw2Ordko/MQSyY4Aq8bhblrGLKoC8ZVdxwnYJl02PicGYHLMbikbGVX9c
mJa/rtTEaw3byyA6gJi17coAfjcKrzPo/FQZdUA3QMWeadhK2rz5NG3suipVKWCo
k9VVds5eJc17dr3NsPttBP74Hz2d1RDy0pIQbd21KIoMlRhDFGi6Lj3UF2mXd9cx
ZOVjsgUCUKYtFlg+0Yciq6ZUq9kq45/Yf+bYBwIDAQABAoIBAQCS5NuS4fGNbmQ0
gI7yRg0poE4wPDO/YjHo2iokMrsjrmYqJc6ry/DaxPLS4pe8F4z3UC1StlBzExKw
+0xNtta8jqPCrAhmBlTS+a9yr5Ey0QtBZf9mgl4ulcPsAT3ZGbaWqmvQRG+Sknve
cptTiZjWVFCl2ZAFcfIbEkADtNmTecHTCxGzd06q44a3lfGPeGQl2JPlEGZ8L/R9
QmxfwdV5Edxm4vgfm9GNGfFiBInh1qIkd7ZnyJtf3WxtFa/Jel7sGPf4pceRrhGH
0aHSYY5aBlZLmLz0sTsyh7AKbBh479P+VW/A2WWQDkN1ERsCtFTstl1ohM5EdcoW
ndfMjQjZAoGBAOtaeXxbfURhNyfbg9y9+bDz+/MO/1BvXhWFSotFOopMKBL0CSWv
JL7eyxwnVPEhJ5Ld1hGIb8b/KdWIiGjICORSwVgv6lzlft5UG6CzDcFdmUMjY9xU
rB8OgR+DHt+rB/hJ+WPHc1Ztmb1TBC/WxWyvwvr+dhASS3sO47sJaQKjAoGBANQt
FEgqQTLhLDZ3miyWQxAyAVCdi+5bgsfPxS7ABpqyeon+nG5COb9Hrz96HExCuxAL
WKd9hLxIYkbsB82paOm6juacpVoSrCK1QQYEgWpy0wRWuhNcwjoMf4WJGQ68ymBG
K8kxsVJ7W1JV44M3lBFSOQp755OmcXNTiMv3SI9NAoGAWg1ANqj7AVMBO6ruhWPb
Si3Q1WuDnU8/fJSHtUpD3+7L1pSxe03MjYvJw3f5NFovPi7LXeIKguXXJ/EZ4J3J
aTOQ7yFGV00oggFEoPRh6v7ZSasc5o0vXqK6HUiaY5KZdhM37Um/g+5jyOEe1P8k
gzWtMUR0ACo/31IPKN2s5GkCgYEAn+Mgh4kf/KFmWd0jFzpcaxXjm8Y9Lm9TTBMr
uiEGWQjqApcVdpj9P0FbtG/mZylaIasLMZwKrH+3X056SubAonDtQqcEi63KfZUs
3MYEaB0DGx/ntLOPwYKtjglUEqD9uDLoyAJkZ42Bsbf7bGQzdiIJzfa0+bTRhWCL
k3hS+3kCgYA6cLb99BRat6WNTxp4YpAa6CiFZRT5U7lTfbXmRqVQT/MPKKWJO3pV
+JdtYnoJZEtUyilqtIh0TsSIcpHMGg9kR7TAeQ9avAJ1OEd7tmhvk4o5VlJd969J
p+wcGu4H7SqlQZI6JZbnYTyF5CPKuzcRcGfDbIp4z3EiV5lRmOdnBQ==
-----END RSA PRIVATE KEY-----
`
	)

	var err error

	block, _ := pem.Decode([]byte(constCaCert))
	certCA, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic(err)
	}

	certClient, err := tls.X509KeyPair([]byte(constClientCert), []byte(constClientKey))
	if err != nil {
		panic(err)
	}

	tlsConfig.RootCAs = x509.NewCertPool()
	tlsConfig.RootCAs.AddCert(certCA)
	tlsConfig.Certificates = []tls.Certificate{certClient}
}

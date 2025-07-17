package testutil

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"strings"
	"time"
)

var (
	clientCaBytes  []byte
	clientKeyBytes []byte
	serverCaBytes  []byte
	serverKeyBytes []byte
	rootCaBytes    []byte
	rootKeyBytes   []byte
)

// 自动生成测试用的 root ca、server ca 和client ca
func init() {
	// 生成根CA
	rootPublicKey, rootPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	rootCert := generateCACert(rootPublicKey, rootPrivateKey, nil, "ED25519 Root CA", true)

	// 生成服务器CA（由根CA签名）
	serverPublicKey, serverPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	serverCert := generateCACert(serverPublicKey, rootPrivateKey, rootCert, "ED25519 Server CA", false)

	// 生成客户端CA（由根CA签名）
	clientPublicKey, clientPrivateKey, _ := ed25519.GenerateKey(rand.Reader)
	clientCert := generateCACert(clientPublicKey, rootPrivateKey, rootCert, "ED25519 Client CA", false)

	// 转换为PEM格式并赋值给全局变量
	rootKeyBytes = encodeED25519PrivateKey(rootPrivateKey)
	rootCaBytes = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootCert.Raw})

	serverKeyBytes = encodeED25519PrivateKey(serverPrivateKey)
	serverCaBytes = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCert.Raw})

	clientKeyBytes = encodeED25519PrivateKey(clientPrivateKey)
	clientCaBytes = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientCert.Raw})
	// 生成证书
	initTLSConfig()
}

// 生成CA证书
func generateCACert(publicKey ed25519.PublicKey, privateKey ed25519.PrivateKey, parent *x509.Certificate, name string, isCA bool) *x509.Certificate {
	template := &x509.Certificate{
		SerialNumber:          randomSerial(),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10年有效期
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.IPv4zero, net.IPv6zero},
		DNSNames:              []string{name, "local"},
		IsCA:                  isCA,
		PublicKey:             publicKey,
	}

	if parent == nil {
		parent = template
	}

	certBytes, _ := x509.CreateCertificate(rand.Reader, template, parent, publicKey, privateKey)
	cert, _ := x509.ParseCertificate(certBytes)
	return cert
}

// 生成随机序列号
func randomSerial() *big.Int {
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	return serial
}

// ED25519私钥编码为PEM格式
func encodeED25519PrivateKey(key ed25519.PrivateKey) []byte {
	keyBytes, _ := x509.MarshalPKCS8PrivateKey(key)
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
}

var (
	// NextProtocols 测试 tls.Config 的 NextProtocols
	NextProtocols = strings.Split("cns test quic quic-go CRYPTO_ERROR", " ")
)

var (
	serverTLSConfig *tls.Config
	clientTLSConfig *tls.Config
)

func initTLSConfig() {
	pb, _ := pem.Decode(rootCaBytes)
	rootCa, err := x509.ParseCertificate(pb.Bytes)
	if err != nil {
		panic(err)
	}
	root, err := x509.SystemCertPool()
	if err != nil {
		panic(err)
	}
	root.AddCert(rootCa)

	serverCert, err := tls.X509KeyPair(serverCaBytes, serverKeyBytes)
	if err != nil {
		panic(err)
	}
	clientCert, err := tls.X509KeyPair(clientCaBytes, clientKeyBytes)
	if err != nil {
		panic(err)
	}

	serverTLSConfig = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    root,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		NextProtos:   NextProtocols,
	}

	clientTLSConfig = &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      root,
		NextProtos:   NextProtocols,
	}
}

// GetTLCConfig 获取测试的 tls 配置对
func GetTLCConfig() (server, client *tls.Config) {
	return serverTLSConfig.Clone(), clientTLSConfig.Clone()
}

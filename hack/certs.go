//go:build ignore

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	patchOnly := len(os.Args) > 1 && os.Args[1] == "--patch-only"

	certPath := filepath.Join("config", "webhook", "certs", "tls.crt")
	keyPath := filepath.Join("config", "webhook", "certs", "tls.key")

	var pemBytes []byte

	if patchOnly {
		var err error
		pemBytes, err = os.ReadFile(certPath)
		if err != nil {
			fmt.Printf("Error reading certificate for patching: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Generating RSA-4096 self-signed certificate for WardK8s Webhook...")
		priv, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			panic(err)
		}

		notBefore := time.Now()
		notAfter := notBefore.Add(365 * 24 * time.Hour)

		serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
		if err != nil {
			panic(err)
		}

		template := x509.Certificate{
			SerialNumber: serialNumber,
			Subject: pkix.Name{
				CommonName: "wardk8s-webhook.wardk8s-system.svc",
			},
			NotBefore:             notBefore,
			NotAfter:              notAfter,
			KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			BasicConstraintsValid: true,
			IsCA:                  true,
			DNSNames: []string{
				"wardk8s-webhook.wardk8s-system.svc",
				"wardk8s-webhook.wardk8s-system.svc.cluster.local",
			},
		}

		derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
		if err != nil {
			panic(err)
		}

		os.MkdirAll(filepath.Join("config", "webhook", "certs"), 0755)

		certOut, err := os.Create(certPath)
		if err != nil {
			panic(err)
		}
		pemBytes = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
		certOut.Write(pemBytes)
		certOut.Close()

		keyOut, err := os.Create(keyPath)
		if err != nil {
			panic(err)
		}
		pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
		keyOut.Close()

		fmt.Printf("Certificates written to %s\n", filepath.Join("config", "webhook", "certs"))
	}

	if patchOnly {
		caBundle := base64.StdEncoding.EncodeToString(pemBytes)
		fmt.Println("Patching ValidatingWebhookConfiguration with new CA Bundle...")
		patch := fmt.Sprintf(`[{"op": "replace", "path": "/webhooks/0/clientConfig/caBundle", "value": "%s"}]`, caBundle)

		cmd := exec.Command("kubectl", "patch", "validatingwebhookconfiguration", "wardk8s-validating-webhook", "--type=json", "-p", patch)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Warning: Could not patch webhook automatically. Error: %v\n", err)
		} else {
			fmt.Println("Webhook patched successfully.")
		}
	}
}

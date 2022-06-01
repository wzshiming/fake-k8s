package pki

import (
	_ "embed"
	"os"
	"path/filepath"
)

// This is just a local key, it doesn't matter if it is leaked.

//go:generate openssl genrsa -out ca.key 2048
//go:generate openssl req -sha256 -x509 -new -nodes -key ca.key -subj "/CN=fake-ca" -out ca.crt -days 365000
//go:generate openssl genrsa -out admin.key 2048
//go:generate openssl req -new -key admin.key -subj "/CN=fake-admin" -out admin.csr -config openssl.cnf
//go:generate openssl x509 -sha256 -req -in admin.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out admin.crt -days 365000 -extensions v3_req -extfile openssl.cnf

var (
	//go:embed ca.crt
	CACrt []byte
	//go:embed admin.key
	AdminKey []byte
	//go:embed admin.crt
	AdminCrt []byte
)

// DumpPki generates a pki directory.
func DumpPki(dir string) error {
	err := os.WriteFile(filepath.Join(dir, "ca.crt"), CACrt, 0644)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(dir, "admin.key"), AdminKey, 0644)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(dir, "admin.crt"), AdminCrt, 0644)
	if err != nil {
		return err
	}
	return nil
}

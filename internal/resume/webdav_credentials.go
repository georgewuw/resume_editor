package resume

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/zalando/go-keyring"
)

const webDAVCredentialService = "resume-editor"

var errWebDAVCredentialNotFound = keyring.ErrNotFound

type webDAVCredentialStore interface {
	Get(service, account string) (string, error)
	Set(service, account, password string) error
	Delete(service, account string) error
}

type systemWebDAVCredentialStore struct{}

func (systemWebDAVCredentialStore) Get(service, account string) (string, error) {
	password, err := keyring.Get(service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", errWebDAVCredentialNotFound
	}
	return password, err
}

func (systemWebDAVCredentialStore) Set(service, account, password string) error {
	return keyring.Set(service, account, password)
}

func (systemWebDAVCredentialStore) Delete(service, account string) error {
	err := keyring.Delete(service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return errWebDAVCredentialNotFound
	}
	return err
}

func webDAVCredentialAccount(config webDAVConfig) string {
	identity := config.ServerURL + "\x00" + config.Username
	digest := sha256.Sum256([]byte(identity))
	return "webdav-" + hex.EncodeToString(digest[:])
}

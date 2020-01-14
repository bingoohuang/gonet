package gonet

import (
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
)

// TempFile ...
func TempFile(content []byte) string {
	if name, err := TempFileE(content); err != nil {
		panic(errors.Wrap(err, "TempFile"))
	} else {
		return name
	}
}

// TempFileE ...
func TempFileE(content []byte) (string, error) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "tmp")
	if err != nil {
		return "", err
	}

	if _, err := tmpFile.Write(content); err != nil {
		return "", err
	}

	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

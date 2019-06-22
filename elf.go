package gonet

import (
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
)

func MustTempFile(content []byte) string {
	if name, err := TempFile(content); err != nil {
		panic(errors.Wrap(err, "TempFile"))
	} else {
		return name
	}
}

func TempFile(content []byte) (string, error) {
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

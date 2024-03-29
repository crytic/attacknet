package pkg

import (
	"errors"
	"github.com/kurtosis-tech/stacktrace"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
)

func WriteFileOnSubpath(subpath, fileName string, file []byte) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(cwd, subpath)

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}

	artifactPath := filepath.Join(path, fileName)
	err = os.WriteFile(artifactPath, file, 0600)
	if err != nil {
		return stacktrace.Propagate(err, "could not write artifacts to %s", artifactPath)
	}
	log.Infof("Wrote test artifact to %s", artifactPath)
	return nil
}

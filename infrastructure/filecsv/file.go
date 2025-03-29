package filecsv

import (
	"os"

	"my-project/infrastructure/logger"
)

func NewFile(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while open file")
		return nil, err
	}

	return file, nil
}

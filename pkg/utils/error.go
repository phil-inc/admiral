package utils

import "github.com/sirupsen/logrus"

func HandleErrorStream(errCh chan error) {
	for err := range errCh {
		logrus.Error(err)
	}
}

package main

import (
	"net/http"

	"github.com/phil-inc/admiral/pkg/backend"
	"github.com/phil-inc/admiral/pkg/backend/gchat"
	"github.com/phil-inc/admiral/pkg/backend/local"
	"github.com/phil-inc/admiral/pkg/backend/loki"
	"github.com/pkg/errors"
)

func InitBackend(logCh chan backend.RawLog, eventCh chan string, errCh chan error, httpCli *http.Client, bType string, url string) error {
	var scopedBackend backend.Backend

	switch bType {

	case "loki":
		scopedBackend = loki.New().Url(url).LogChannel(logCh).ErrChannel(errCh).Client(httpCli).Build()

	case "gchat":
		scopedBackend = gchat.New().Url(url).TextChannel(eventCh).ErrChannel(errCh).Client(httpCli).Build()

	case "local":
		backendBuilder := local.New()

		if logCh != nil {
			backendBuilder = backendBuilder.LogChannel(logCh)
		}

		if eventCh != nil {
			backendBuilder = backendBuilder.EventChannel(eventCh)
		}

		scopedBackend = backendBuilder.ErrChannel(errCh).Build()

	case "":
		break

	default:
		return errors.Errorf("invalid type in backend: %s", bType)
	}

	if scopedBackend != nil {
		go scopedBackend.Stream()
	}
	return nil
}

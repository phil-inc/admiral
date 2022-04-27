package target

import (
	"net/http"

	"github.com/phil-inc/admiral/config"
)

type Target interface {
	Init(conf *config.Config) error
	Test(c http.Client, appName string) error
}

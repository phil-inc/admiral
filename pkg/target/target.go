package target

import (
	"net/http"

	"github.com/phil-inc/admiral/config"
)

type TargetParams struct {
	HttpClient http.Client
	ApiKeys    map[string]string
}

type Target interface {
	Init(conf *config.Config) error
	InitParams(params TargetParams) error
	Test(appName string) error
}

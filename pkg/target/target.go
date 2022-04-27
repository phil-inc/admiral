package target

import "github.com/phil-inc/admiral/config"

type Target interface {
	Init(conf *config.Config) error
	Test(appName string) error
}

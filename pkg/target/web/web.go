package web

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/phil-inc/admiral/config"
)

type Web struct {
	url string
}

// Init creates the loki configuration
func (l *Web) Init(c *config.Config) error {
	url := c.Performance.Target.Web.Url
	l.url = url

	return nil
}

// Init creates the loki configuration
func (l *Web) Test(c http.Client, appName string) error {

	// TODO: Send web page test request here.
	// TODO: Print out result

	apiKey := os.Getenv("WEB_PERFORMANCE_API_KEY")
	url := fmt.Sprintf("https://www.webpagetest.org/runtest.php?k=%s&url=%s&f=json", apiKey, l.url)
	resp, err := c.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Body)

	return nil
}

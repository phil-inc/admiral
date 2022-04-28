package web

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/target"
	"github.com/sirupsen/logrus"
)

const runWebPageTestUrl = "https://www.webpagetest.org/runtest.php"
const checkWebPageTestUrl = "https://www.webpagetest.org/jsonResult.php"

type Web struct {
	tests  map[string][]config.Test
	http   http.Client
	apiKey string
}

// Init creates the configuration for web performance testing
func (w *Web) Init(c *config.Config) error {
	tests := c.Performance.Target.Web.Tests
	w.tests = tests

	return nil
}

// InitParams sets additiona; configuration to run web performance testing.
func (w *Web) InitParams(targetParams target.TargetParams) error {
	w.http = targetParams.HttpClient
	w.apiKey = targetParams.ApiKeys["web"]

	return nil
}

// Test runs web performance against a list of urls.
func (w *Web) Test(appLabel string) error {
	appTests := w.tests[appLabel]

	testIds := []string{}
	for _, test := range appTests {
		testId := w.runTest(test)
		testIds = append(testIds, testId)
		logrus.Printf("Received web performance test id for %s: %s", appLabel, testId)
	}

	logrus.Printf("[performance] Received test IDs for %s: %s", appLabel, testIds)

	for _, testId := range testIds {
		statusCh := make(chan struct{})
		go w.checkStatus(statusCh, testId)

		<-statusCh

		out, _ := json.Marshal(statusCh)
		logrus.Printf("[performance] Received test results for %s: %s", appLabel, string(out))
	}

	return nil
}

func (w *Web) runTest(test config.Test) string {
	url := fmt.Sprintf("%s?k=%s&url=%s&mobile=%d&f=json", runWebPageTestUrl, w.apiKey, test.Url, test.Mobile)
	resp, err := w.http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// TODO Need to validate the actual reponse body.
	var responseData struct {
		data struct {
			testId string
		}
	}
	json.Unmarshal(body, &responseData)

	return responseData.data.testId
}

func (w *Web) checkStatus(statusCh chan struct{}, testId string) {
	for {
		url := fmt.Sprintf("%s?test=%s", checkWebPageTestUrl, testId)
		resp, err := w.http.Get(url)
		if err != nil {
			log.Fatal(err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		var responseData struct {
			data       struct{}
			statusCode int
		}
		json.Unmarshal(body, &responseData)

		// TODO: Maybe wrap the reponse into a success or fail structs?
		switch {
		case responseData.statusCode == 200:
			statusCh <- responseData.data
			break
		case responseData.statusCode >= 400:
			log.Fatal(fmt.Errorf("Unable to get results for test: %s", testId))
			var empty struct{}
			statusCh <- empty
			break
		}
	}
}

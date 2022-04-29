package web

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

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
		testId := w.runTest(appLabel, test)
		testIds = append(testIds, testId)
		logrus.Printf("[performance][%s] Ran webpagetest: url=%s | testId=%s", appLabel, test.Url, testId)
	}

	for _, testId := range testIds {
		statusCh := make(chan struct{})
		go w.checkStatus(statusCh, appLabel, testId)

		go func(testId string) {
			<-statusCh

			output, err := json.Marshal(statusCh)
			if err != nil {
				logrus.Errorf("[performance][%s] Error marshaling status check: testId=%s | error=%s", appLabel, testId, err)
			}
			logrus.Printf("[performance][%s] received webpagetest result: testId=%s | result=%s", appLabel, testId, string(output))
		}(testId)
	}

	return nil
}

func (w *Web) runTest(appLabel string, test config.Test) string {
	url := fmt.Sprintf("%s?k=%s&url=%s&mobile=%d&runs=%d&f=json", runWebPageTestUrl, w.apiKey, test.Url, test.Mobile, test.Runs)
	resp, err := w.http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("[performance][%s] Error reading body from webpagetest result: url=%s | error=%s", appLabel, test.Url, err)
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

func (w *Web) checkStatus(statusCh chan struct{}, appLabel string, testId string) {
	for {
		url := fmt.Sprintf("%s?test=%s", checkWebPageTestUrl, testId)
		resp, err := w.http.Get(url)
		if err != nil {
			log.Fatal(err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logrus.Errorf("[performance][%s] Error reading body from webpagetest result: testId=%s | error=%s", appLabel, testId, err)
		}

		var responseData struct {
			data       struct{}
			statusCode int
		}
		json.Unmarshal(body, &responseData)

		// TODO: Should we make a distinction between failed and successful test runs?
		switch {
		case responseData.statusCode >= 200:
			statusCh <- responseData.data
			return
		}

		time.Sleep(3000 * time.Second)
	}
}

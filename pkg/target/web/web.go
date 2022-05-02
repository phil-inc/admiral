package web

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/phil-inc/admiral/config"
	"github.com/phil-inc/admiral/pkg/target"
	"github.com/sirupsen/logrus"
)

const RUN_WEB_PAGE_TEST_URL = "https://www.webpagetest.org/runtest.php"
const CHECK_WEB_PAGE_TEST_URL = "https://www.webpagetest.org/jsonResult.php"
const MAX_RETRY = 3

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

// InitParams sets additional configuration to run web performance testing.
func (w *Web) InitParams(targetParams target.TargetParams) error {
	w.http = targetParams.HttpClient
	w.apiKey = targetParams.ApiKeys["web"]

	return nil
}

// Test runs web performance against a list of urls.
func (w *Web) Test(appLabel string) error {
	appTests := w.tests[appLabel]

	testIds := []string{}
	// Run web performance tests.
	for _, test := range appTests {
		testId := w.runTest(appLabel, test)
		testIds = append(testIds, testId)
		logrus.Printf("[performance][%s] Ran webpagetest: url=%s | testId=%s", appLabel, test.Url, testId)
	}

	// Check the status for each test given a testId.
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
	var responseData struct {
		data struct {
			testId string
		}
	}
	retryCount := 0
	url := fmt.Sprintf("%s?k=%s&url=%s&mobile=%d&runs=%d&f=json", RUN_WEB_PAGE_TEST_URL, w.apiKey, test.Url, test.Mobile, test.Runs)
	for retryCount <= MAX_RETRY {
		resp, err := w.http.Get(url)
		if err != nil {
			retryCount++
			logrus.Errorf("[performance][%s] Error running webpagetest: url=%s | retryCount=%d | error=%s", appLabel, test.Url, retryCount, err)
			time.Sleep(time.Duration(retryCount) * time.Second)
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logrus.Errorf("[performance][%s] Error reading webpagetest result: url=%s | error=%s", appLabel, test.Url, err)
			}
			json.Unmarshal(body, &responseData)
			break
		}
    }
	return responseData.data.testId
}

func (w *Web) checkStatus(statusCh chan struct{}, appLabel string, testId string) {
	var responseData struct {
		data       struct{}
		statusCode int
	}
	retryCount := 0
	url := fmt.Sprintf("%s?test=%s", CHECK_WEB_PAGE_TEST_URL, testId)
	for retryCount <= MAX_RETRY {
		resp, err := w.http.Get(url)
		if err != nil {
			retryCount++
			logrus.Errorf("[performance][%s] Error checking webpagetest: testId=%s | error=%s", appLabel, testId, err)
			time.Sleep(time.Duration(retryCount) * time.Second)
		} else {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logrus.Errorf("[performance][%s] Error reading webpagetest result: testId=%s | error=%s", appLabel, testId, err)
			}
			json.Unmarshal(body, &responseData)

			switch responseData.statusCode {
			case 200:
				statusCh <- responseData.data
				return
			case 102, 400, 401:
				retryCount++
				time.Sleep(time.Duration(retryCount) * time.Second)
				break
			}
		}
    }
	statusCh <- responseData.data
}

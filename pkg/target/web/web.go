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

type QueueResponse struct {
	data struct {
		testId string
	}
	statusCode int
}

type ResultsResponse struct {
	data       TestResult
	statusCode int
}

type TestResult struct {
	id           string
	url          string
	connectivity string
	location     string
	mobile       int
	average      struct {
		firstView  TestViewResult
		repeatView TestViewResult
	}
}

type TestViewResult struct {
	loadTime               float32
	bytesOut               float32
	bytesIn                float32
	fullyLoaded            float32
	requests               int
	responses_200          int
	responses_404          int
	responses_other        int
	cached                 int
	connections            int
	firstPaint             float32
	firstContentfulPaint   float32
	firstImagePaint        float32
	domInteractive         float32
	domElements            int
	renderBlockingCSS      int
	renderBlockingJS       int
	score_cache            int
	score_cdn              int
	score_gzip             int
	score_cookies          int
	score_minify           int
	score_compress         int
	score_etags            int
	score_progressive_jpeg int
	visualComplete         float32
	run                    int
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
	logrus.Printf("[performance][%s] Initiated web performance test.", appLabel)

	// Run web performance tests.
	testIds := []string{}
	for _, test := range appTests {
		testId, err := w.runTestWithRetries(appLabel, test)
		if err != nil {
			logrus.Errorf("[performance][%s] Error running test | testUrl =%s | error=%s ", appLabel, test.Url, err)
			continue
		}
		testIds = append(testIds, testId)
	}

	// Get result for each testId.
	for _, testId := range testIds {
		resultsCh := make(chan TestResult)
		go w.getResultsWithRetries(resultsCh, appLabel, testId)

		go func(testId string) {
			<-resultsCh
			output, err := json.Marshal(resultsCh)
			if err != nil {
				logrus.Errorf("[performance][%s] Error marshaling test result: testId=%s | error=%s", appLabel, testId, err)
			}
			logrus.Printf("[performance][%s] Received test result: testId=%s | result=%s", appLabel, testId, string(output))
		}(testId)
	}

	return nil
}

func (w *Web) runTestWithRetries(appLabel string, test config.Test) (testId string, err error) {
	retry := 0
	for retry <= MAX_RETRY {
		resp, err := w.queueTest(appLabel, test)
		if err != nil {
			retry++
			logrus.Errorf("[performance][%s] Error queueing test | retry=%d | error=%s", appLabel, retry, err)
			time.Sleep(time.Duration(retry) * time.Second)
		}

		respObj := w.constructQueueResponse(appLabel, resp)
		testId, err = w.handleQueueResponse(appLabel, respObj)
		if err != nil {
			retry++
			logrus.Errorf("[performance][%s] Error handling queue response | retry=%d | error=%s", appLabel, retry, err)
			time.Sleep(time.Duration(retry) * time.Second)
		}
	}
	return testId, err
}

func (w *Web) queueTest(appLabel string, test config.Test) (resp *http.Response, err error) {
	url := fmt.Sprintf("%s?k=%s&url=%s&mobile=%d&runs=%d&f=json", RUN_WEB_PAGE_TEST_URL, w.apiKey, test.Url, test.Mobile, test.Runs)

	resp, err = w.http.Get(url)
	if err != nil {
		logrus.Errorf("[performance][%s] Error queuing webpagetest: url=%s | error=%s", appLabel, url, err)
	}

	return resp, err
}

func (w *Web) constructQueueResponse(appLabel string, resp *http.Response) (respObj QueueResponse) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("[performance][%s] Error converting to queue response | error=%s", appLabel, err)
	}
	json.Unmarshal(body, &respObj)

	return respObj
}

func (w *Web) handleQueueResponse(appLabel string, respObj QueueResponse) (testId string, err error) {
	switch respObj.statusCode {
	case 200:
		testId = respObj.data.testId
		logrus.Printf("[performance][%s] Successfully queued test: | testId=%s", appLabel, testId)
	default:
		err = fmt.Errorf("[performance][%s] Failed to get test ID:  statusCode=%d", appLabel, respObj.statusCode)
	}
	return testId, err
}

func (w *Web) getResultsWithRetries(resultsCh chan TestResult, appLabel string, testId string) {
	var results TestResult
	retry := 0
	for retry <= MAX_RETRY {
		resp, err := w.checkStatus(appLabel, testId)
		if err != nil {
			retry++
			logrus.Errorf("[performance][%s] Error queueing test | retry=%d | error=%s", appLabel, retry, err)
			time.Sleep(time.Duration(retry) * time.Second)
		}

		respObj := w.constructToResultsResponse(appLabel, resp)
		results, err = w.handleResultsResponse(appLabel, respObj)

		if err != nil {
			retry++
			logrus.Errorf("[performance][%s] Error handling queue response | retry=%d | error=%s", appLabel, retry, err)
			time.Sleep(time.Duration(retry) * time.Second)
		}

	}
	resultsCh <- results
}

func (w *Web) checkStatus(appLabel string, testId string) (resp *http.Response, err error) {
	url := fmt.Sprintf("%s?test=%s", CHECK_WEB_PAGE_TEST_URL, testId)
	resp, err = w.http.Get(url)
	if err != nil {
		logrus.Errorf("[performance][%s] Error queuing webpagetest: url=%s | error=%s", appLabel, url, err)
	}

	return resp, err
}

func (w *Web) constructToResultsResponse(appLabel string, resp *http.Response) (respObj ResultsResponse) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("[performance][%s] Error converting to results response | error=%s", appLabel, err)
	}
	json.Unmarshal(body, &respObj)

	return respObj
}

func (w *Web) handleResultsResponse(appLabel string, respObj ResultsResponse) (result TestResult, err error) {
	switch respObj.statusCode {
	case 100, 101:
	case 200:
		result = respObj.data
		logrus.Printf("[performance][%s] Successfully received test result: | testId=%s", appLabel, result.id)
	default:
		err = fmt.Errorf("[performance][%s] Failed to get test result:  statusCode=%d", appLabel, respObj.statusCode)

	}
	return result, err
}

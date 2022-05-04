package web

import (
	"encoding/json"
	"fmt"
	"io"
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
	Data       QueueData
	StatusCode int
}

type QueueData struct {
	TestId string
}

type ResultsResponse struct {
	Data       ResultData
	StatusCode int
	StatusText string
}

type ResultData struct {
	Id           string
	Url          string
	Connectivity string
	Location     string
	Mobile       int
	Average      AverageResult
}

type AverageResult struct {
	FirstView  ViewResult
	RepeatView ViewResult
}

type ViewResult struct {
	LoadTime               float32
	BytesOut               float32
	BytesIn                float32
	FullyLoaded            float32
	Requests               int
	Responses_200          int
	Responses_404          int
	Responses_other        int
	Cached                 int
	Connections            int
	FirstPaint             float32
	FirstContentfulPaint   float32
	FirstImagePaint        float32
	DomInteractive         float32
	DomElements            int
	RenderBlockingCSS      int
	RenderBlockingJS       int
	Score_cache            int
	Score_cdn              int
	Score_gzip             int
	Score_cookies          int
	Score_minify           int
	Score_compress         int
	Score_etags            int
	Score_progressive_jpeg int
	VisualComplete         float32
	Run                    int
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
	logrus.Printf("[performance][%s] Initiated test.", appLabel)

	appTests := w.tests[appLabel]

	// Run web performance tests.
	testIds := []string{}
	for i, test := range appTests {
		testId, err := w.runTestWithRetries(appLabel, test)
		if err != nil {
			logrus.Errorf("[performance][%s] Skipping test: index=%d | testUrl=%s.", appLabel, i, test.Url)
			continue
		}
		testIds = append(testIds, testId)
	}

	time.Sleep(5 * time.Second)

	// Get result for each testId.
	for _, testId := range testIds {
		resultsCh := make(chan ResultData)
		go w.getResultsWithRetries(resultsCh, appLabel, testId)

		go func(testId string) {
			result := <-resultsCh
			resultOutput, err := json.Marshal(result)
			if err != nil {
				logrus.Errorf("[performance][%s] Error marshaling test result: testId=%s | error=%s", appLabel, testId, err)
			}
			logrus.Printf("[performance][%s] Successfully received test result: testId=%s | result=%s", appLabel, testId, string(resultOutput))
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
			time.Sleep(time.Duration(retry) * time.Second)
			continue
		}

		respObj := w.constructQueueResponse(appLabel, resp.Body)
		testId, err = w.handleQueueResponse(appLabel, respObj)

		if err != nil {
			retry++
			time.Sleep(time.Duration(retry) * time.Second)
			continue
		}
		break
	}
	return
}

func (w *Web) queueTest(appLabel string, test config.Test) (resp *http.Response, err error) {
	url := fmt.Sprintf("%s?k=%s&url=%s&mobile=%d&runs=%d&f=json", RUN_WEB_PAGE_TEST_URL, w.apiKey, test.Url, test.Mobile, test.Runs)
	resp, err = w.http.Get(url)
	if err != nil {
		logrus.Errorf("[performance][%s] Error queuing test: url=%s | error=%s", appLabel, url, err)
	}
	return
}

func (w *Web) constructQueueResponse(appLabel string, responseBody io.ReadCloser) (respObj QueueResponse) {
	defer responseBody.Close()
	body, err := ioutil.ReadAll(responseBody)
	if err != nil {
		logrus.Errorf("[performance][%s] Error reading queue response body | error=%s", appLabel, err)
	}

	err = json.Unmarshal(body, &respObj)
	if err != nil {
		logrus.Errorf("[performance][%s] Error unmarshalling queue response body | error=%s", appLabel, err)
	}
	return
}

func (w *Web) handleQueueResponse(appLabel string, respObj QueueResponse) (testId string, err error) {
	switch respObj.StatusCode {
	case 200:
		testId = respObj.Data.TestId
		logrus.Printf("[performance][%s] Successfully queued test: testId=%s", appLabel, testId)
	default:
		err = fmt.Errorf("Could not queue test: statusCode=%d", respObj.StatusCode)
		logrus.Errorf("[performance][%s] %s", appLabel, err)
	}
	return
}

func (w *Web) getResultsWithRetries(resultsCh chan ResultData, appLabel string, testId string) {
	var result ResultData
	retry := 0
	for retry <= MAX_RETRY {
		resp, err := w.checkStatus(appLabel, testId)
		if err != nil {
			retry++
			time.Sleep(time.Duration(retry) * time.Second)
			continue
		}

		respObj := w.constructToResultsResponse(appLabel, resp.Body)
		result, err = w.handleResultsResponse(appLabel, respObj)

		if result == (ResultData{}) {
			if err != nil {
				retry++ // Increase retry if failed to get test results.
				time.Sleep(time.Duration(retry) * time.Second)
			}
			continue
		}
		break
	}
	resultsCh <- result
}

func (w *Web) checkStatus(appLabel string, testId string) (resp *http.Response, err error) {
	url := fmt.Sprintf("%s?test=%s", CHECK_WEB_PAGE_TEST_URL, testId)
	resp, err = w.http.Get(url)
	if err != nil {
		logrus.Errorf("[performance][%s] Error queuing webpagetest: url=%s | error=%s", appLabel, url, err)
	}
	return
}

func (w *Web) constructToResultsResponse(appLabel string, responseBody io.ReadCloser) (respObj ResultsResponse) {
	defer responseBody.Close()
	body, err := ioutil.ReadAll(responseBody)
	if err != nil {
		logrus.Errorf("[performance][%s] Error converting to results response | error=%s", appLabel, err)
	}

	err = json.Unmarshal(body, &respObj)
	if err != nil {
		logrus.Errorf("[performance][%s] Error unmarshalling queue response body | error=%s", appLabel, err)
	}
	return
}

func (w *Web) handleResultsResponse(appLabel string, respObj ResultsResponse) (result ResultData, err error) {
	switch respObj.StatusCode {
	case 100, 101:
		logrus.Printf("[performance][%s] Web performance test is still in progress: testId=%s.", appLabel, respObj.Data.Id)
		time.Sleep(5 * time.Second)
	case 200:
		result = respObj.Data
	default:
		err = fmt.Errorf("Could not get test result: statusCode=%d", respObj.StatusCode)
		logrus.Errorf("[performance][%s] %s", appLabel, err)
	}
	return
}

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
	TestID string
}

type ResultsResponse struct {
	Data       ResultData
	StatusCode int
	StatusText string
}

type ResultData struct {
	ID           string
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
			logrus.Errorf("[performance][%s] Skipping test. index=%d | testUrl=%s.", appLabel, i, test.Url)
			continue
		}
		testIds = append(testIds, testId)
	}

	time.Sleep(30 * time.Second)

	// Get result for each testId.
	for _, testId := range testIds {
		resultsCh := make(chan ResultData)
		go w.getResultsWithRetries(resultsCh, appLabel, testId)

		go func(testId string) {
			result := <-resultsCh
			resultOutput, err := json.Marshal(result)
			if err != nil {
				logrus.Errorf("[performance][%s] Error marshaling test result. testId=%s | error=%s", appLabel, testId, err)
			}

			var resultData ResultData
			err = json.Unmarshal(resultOutput, &resultData)
			if err != nil {
				logrus.Errorf("[performance][%s] Error unmarshaling test result. testId=%s | error=%s", appLabel, testId, err)
			}

			if resultData.ID != "" {
				logrus.Printf("[performance][%s] Received test result. testId=%s | result=[%s]", appLabel, testId, w.formatResultsResponse(resultData))
			}
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
		logrus.Errorf("[performance][%s] Call failed to queue test. method=GET | url=%s | error=%s", appLabel, url, err)
	}
	return
}

func (w *Web) constructQueueResponse(appLabel string, responseBody io.ReadCloser) (respObj QueueResponse) {
	defer responseBody.Close()
	body, err := ioutil.ReadAll(responseBody)
	if err != nil {
		logrus.Errorf("[performance][%s] Error reading queue response body. error=%s", appLabel, err)
	}

	err = json.Unmarshal(body, &respObj)
	if err != nil {
		logrus.Errorf("[performance][%s] Error unmarshalling queue response body. error=%s", appLabel, err)
	}
	return
}

func (w *Web) handleQueueResponse(appLabel string, respObj QueueResponse) (testId string, err error) {
	switch respObj.StatusCode {
	case 200:
		testId = respObj.Data.TestID
		logrus.Printf("[performance][%s] Queued test. testId=%s", appLabel, testId)
	default:
		err = fmt.Errorf("Could not queue test. statusCode=%d", respObj.StatusCode)
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
		result, err = w.handleResultsResponse(appLabel, testId, respObj)

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
		logrus.Errorf("[performance][%s] Call failed to check test status. method=GET | url=%s | error=%s", appLabel, url, err)
	}
	return
}

func (w *Web) constructToResultsResponse(appLabel string, responseBody io.ReadCloser) (respObj ResultsResponse) {
	defer responseBody.Close()
	body, err := ioutil.ReadAll(responseBody)
	if err != nil {
		logrus.Errorf("[performance][%s] Error converting to results response. error=%s", appLabel, err)
	}

	err = json.Unmarshal(body, &respObj)
	if err != nil {
		logrus.Errorf("[performance][%s] Error unmarshalling queue response body. error=%s", appLabel, err)
	}
	return
}

func (w *Web) handleResultsResponse(appLabel, testId string, respObj ResultsResponse) (result ResultData, err error) {
	switch respObj.StatusCode {
	case 100, 101:
		logrus.Printf("[performance][%s] Test still in progress. testId=%s", appLabel, testId)
		time.Sleep(60 * time.Second)
	case 200:
		result = respObj.Data
	default:
		err = fmt.Errorf("Could not get test result. testId=%s | statusCode=%d", testId, respObj.StatusCode)
		logrus.Errorf("[performance][%s] %s", appLabel, err)
	}
	return
}

func (w *Web) formatResultsResponse(resultData ResultData) string {
	url := fmt.Sprintf("url=%s", resultData.Url)
	mobile := fmt.Sprintf("| mobile=%d", resultData.Mobile)
	connectivity := fmt.Sprintf("| connectivity=%s", resultData.Connectivity)
	location := fmt.Sprintf("| location=%s", resultData.Location)
	loadTime := fmt.Sprintf("| loadTime=%.1f(F),%.1f(R)", resultData.Average.FirstView.LoadTime, resultData.Average.RepeatView.LoadTime)
	visualComplete := fmt.Sprintf("| visualComplete=%.1f(F),%.1f(R)", resultData.Average.FirstView.VisualComplete, resultData.Average.RepeatView.VisualComplete)
	run := fmt.Sprintf("| run=%d(F),%d(R)", resultData.Average.FirstView.Run, resultData.Average.RepeatView.Run)
	general := fmt.Sprintf("%s %s %s %s %s %s %s", url, mobile, connectivity, location, loadTime, visualComplete, run)

	bytesOut := fmt.Sprintf("| bytesOut=%.1f(F),%.1f(R)", resultData.Average.FirstView.BytesOut, resultData.Average.RepeatView.BytesOut)
	bytesIn := fmt.Sprintf("| bytesIn=%.1f(F),%.1f(R)", resultData.Average.FirstView.BytesIn, resultData.Average.RepeatView.BytesIn)
	bytesInfo := fmt.Sprintf("%s %s", bytesOut, bytesIn)

	domInteractive := fmt.Sprintf("| domInteractive=%.1f(F),%.1f(R)", resultData.Average.FirstView.DomInteractive, resultData.Average.RepeatView.DomInteractive)
	domElements := fmt.Sprintf("| domElements=%d(F),%d(R)", resultData.Average.FirstView.DomElements, resultData.Average.RepeatView.DomElements)
	domInfo := fmt.Sprintf("%s %s", domElements, domInteractive)

	firstPaint := fmt.Sprintf("| firstPaint=%.1f(F),%.1f(R)", resultData.Average.FirstView.FirstPaint, resultData.Average.RepeatView.FirstPaint)
	firstContentfulPaint := fmt.Sprintf("| firstContentfulPaint=%.1f(F),%.1f(R)", resultData.Average.FirstView.FirstContentfulPaint, resultData.Average.RepeatView.FirstContentfulPaint)
	firstImagePaint := fmt.Sprintf("| firstImagePaint=%.1f(F),%.1f(R)", resultData.Average.FirstView.FirstImagePaint, resultData.Average.RepeatView.FirstImagePaint)
	firstPaints := fmt.Sprintf("%s %s %s", firstPaint, firstContentfulPaint, firstImagePaint)

	renderBlockingCSS := fmt.Sprintf("| renderBlockingCSS=%d(F),%d(R)", resultData.Average.FirstView.RenderBlockingCSS, resultData.Average.RepeatView.RenderBlockingCSS)
	renderBlockingJS := fmt.Sprintf("| renderBlockingJS=%d(F),%d(R)", resultData.Average.FirstView.RenderBlockingJS, resultData.Average.RepeatView.RenderBlockingJS)
	renderBlocking := fmt.Sprintf("%s %s", renderBlockingCSS, renderBlockingJS)

	requests := fmt.Sprintf("| requests=%d(F),%d(R)", resultData.Average.FirstView.Requests, resultData.Average.RepeatView.Requests)

	responses_200 := fmt.Sprintf("| responses_200=%d(F),%d(R)", resultData.Average.FirstView.Responses_200, resultData.Average.RepeatView.Responses_200)
	responses_404 := fmt.Sprintf("| responses_404=%d(F),%d(R)", resultData.Average.FirstView.Responses_404, resultData.Average.RepeatView.Responses_404)
	responses_other := fmt.Sprintf("| responses_other=%d(F),%d(R)", resultData.Average.FirstView.Responses_other, resultData.Average.RepeatView.Responses_other)
	responses := fmt.Sprintf("%s %s %s", responses_200, responses_404, responses_other)

	score_cache := fmt.Sprintf("| score_cache=%d(F),%d(R)", resultData.Average.FirstView.Score_cache, resultData.Average.RepeatView.Score_cache)
	score_cdn := fmt.Sprintf("| score_cdn=%d(F),%d(R)", resultData.Average.FirstView.Score_cdn, resultData.Average.RepeatView.Score_cdn)
	score_gzip := fmt.Sprintf("| score_gzip=%d(F),%d(R)", resultData.Average.FirstView.Score_gzip, resultData.Average.RepeatView.Score_gzip)
	score_cookies := fmt.Sprintf("| score_cookies=%d(F),%d(R)", resultData.Average.FirstView.Score_cookies, resultData.Average.RepeatView.Score_cookies)
	score_minify := fmt.Sprintf("| score_minify=%d(F),%d(R)", resultData.Average.FirstView.Score_minify, resultData.Average.RepeatView.Score_minify)
	score_compress := fmt.Sprintf("| score_compress=%d(F),%d(R)", resultData.Average.FirstView.Score_compress, resultData.Average.RepeatView.Score_compress)
	score_etags := fmt.Sprintf("| score_etags=%d(F),%d(R)", resultData.Average.FirstView.Score_etags, resultData.Average.RepeatView.Score_etags)
	score_progressive_jpeg := fmt.Sprintf(" | score_progressive_jpeg=%d(F),%d(R)", resultData.Average.FirstView.Score_progressive_jpeg, resultData.Average.RepeatView.Score_progressive_jpeg)
	scores := fmt.Sprintf("%s %s %s %s %s %s %s %s", score_cache, score_cdn, score_gzip, score_cookies, score_minify, score_compress, score_etags, score_progressive_jpeg)

	return fmt.Sprintf("%s %s %s %s %s %s %s %s", general, bytesInfo, domInfo, firstPaints, renderBlocking, requests, responses, scores)
}

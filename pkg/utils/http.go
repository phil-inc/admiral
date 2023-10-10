package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/exp/slices"
)

var SUCCESSFUL_STATUS_CODES = []int{200, 201, 202, 203, 204, 205, 206, 207, 208, 226}

func Send(data interface{}, method string, url string, client *http.Client) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	if !slices.Contains(SUCCESSFUL_STATUS_CODES, res.StatusCode) {
		return fmt.Errorf("%s - %s", res.Status, res.Body)
	}

	return nil
}

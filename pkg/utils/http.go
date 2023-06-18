package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

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

	if res.StatusCode == 400 {
		return errors.New(res.Status)
	}

	return nil
}

package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
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
		return fmt.Errorf("%s - %s", res.Status, res.Body)
	}

	return nil
}

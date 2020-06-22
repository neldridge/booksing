package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/gnur/booksing"
	log "github.com/sirupsen/logrus"
)

func uploadFile(url, filepath string) error {
	data, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer data.Close()

	req, err := http.NewRequest("PUT", url, data)
	if err != nil {
		return err
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	a, err := data.Stat()
	req.ContentLength = a.Size()
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		fmt.Println(res.Status)
		a, _ := ioutil.ReadAll(res.Body)
		fmt.Println(string(a))
		return errors.New("not working bitches")
	}
	return nil
}

func downloadFile(url, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	_, err = io.Copy(file, res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("Got a %d response code", res.StatusCode)
	}

	return nil
}

func (cfg *configuration) addToBooksing(c booksing.ConvertRequest) error {
	url := fmt.Sprintf("%s/api/book/%s/%s",
		cfg.BooksingHost,
		c.Hash,
		c.TargetFormat)

	in := c.Loc

	js, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(js))
	if err != nil {
		// return true to be safe
		return err
	}
	client := &http.Client{}
	req.Header.Add("x-api-key", cfg.BooksingAPIKey)
	req.Header.Add("Contenty-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 403 {
			log.Fatal("access denied")
		}
		return err
	}
	resp.Body.Close()

	return nil
}

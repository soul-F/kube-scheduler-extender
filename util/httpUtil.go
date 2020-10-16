package util

import (
	"bytes"
	mytls "crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	url2 "net/url"
	"strconv"
	"strings"
	"time"
)

func DoRequest(tls bool, requestType, url, requestParam, requestHeader, requestBody string, requestTimeout time.Duration) (err error, result string) {
	var client http.Client
	if tls {
		conf := &mytls.Config{
			InsecureSkipVerify: true,
		}
		tr := &http.Transport{
			TLSClientConfig:   conf,
			DisableKeepAlives: true,
		}
		client = http.Client{Transport: tr, Timeout: requestTimeout}
	} else {
		tr := &http.Transport{
			DisableKeepAlives: true,
		}
		client = http.Client{Transport: tr, Timeout: requestTimeout}
	}
	if requestType == "GET" {
		req, _ := http.NewRequest("GET", url, nil)
		if requestHeader != "" {
			for _, v := range strings.Split(requestHeader, "&") {
				if v != "" {
					item := strings.Split(v, "=")
					req.Header.Add(item[0], item[1])
				}
			}
		}

		resp, err1 := client.Do(req) //发送请求
		if err1 == nil {
			defer resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode <= 206 {
				result, _ := ioutil.ReadAll(resp.Body)
				responseString := string(result)
				return nil, responseString
			}
			return errors.New("访问出错，错误码为" + strconv.Itoa(resp.StatusCode)), ""
		}
		return err1, ""

	}
	req, _ := http.NewRequest(requestType, url, nil)
	if requestBody != "" {
		body := bytes.NewBuffer([]byte(requestBody))
		req, _ = http.NewRequest(requestType, url, body)
	}
	if requestHeader != "" {
		for _, v := range strings.Split(requestHeader, "&") {
			if v != "" {
				item := strings.Split(v, "=")
				req.Header.Add(item[0], item[1])
			}
		}
	}
	if requestParam != "" {
		for _, v := range strings.Split(requestParam, "&") {
			if v != "" {
				item := strings.Split(v, "=")
				req.Form.Add(item[0], item[1])
			}
		}
	}

	resp, err := client.Do(req) //发送请求
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode <= 206 {
			result, _ := ioutil.ReadAll(resp.Body)
			responseString := string(result)
			return nil, responseString
		}
		return errors.New("访问出错，错误码为" + strconv.Itoa(resp.StatusCode)), ""
	}
	return err, ""

}

func GetResponse(requestType, url, requestParam, requestHeader, requestBody string, timeout time.Duration, cookies []*http.Cookie) (*http.Response, error) {
	var client http.Client
	tr := &http.Transport{
		DisableKeepAlives: true,
	}
	client = http.Client{
		Transport: tr,
		Timeout:   timeout,
	}
	if requestType == "GET" {
		req, _ := http.NewRequest("GET", url, nil)
		if requestHeader != "" {
			for _, v := range strings.Split(requestHeader, "&") {
				if v != "" {
					item := strings.Split(v, "=")
					req.Header.Add(item[0], item[1])
				}
			}
		}
		if cookies != nil {
			for _, v := range cookies {
				req.AddCookie(v)
			}
		}

		resp, err1 := client.Do(req) //发送请求

		return resp, err1
	}

	req, err := http.NewRequest(requestType, url, nil)
	if err != nil {
		fmt.Println(err)
	}
	if requestBody != "" {
		body := bytes.NewBuffer([]byte(requestBody))
		req, _ = http.NewRequest(requestType, url, body)
	}
	if requestHeader != "" {
		for _, v := range strings.Split(requestHeader, "&") {
			if v != "" {
				item := strings.Split(v, "=")
				req.Header.Add(item[0], item[1])
			}
		}
	}
	if requestParam != "" {
		req.Form = make(url2.Values)
		for _, v := range strings.Split(requestParam, "&") {
			if v != "" {
				item := strings.Split(v, "=")
				req.Form.Add(item[0], item[1])
			}
		}
	}
	if cookies != nil {
		for _, v := range cookies {
			req.AddCookie(v)
		}
	}

	resp, err := client.Do(req) //发送请求
	return resp, err
}

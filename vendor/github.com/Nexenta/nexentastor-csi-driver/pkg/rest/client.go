package rest

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const requestTimeout = 30 * time.Second

// Client - request client for any REST API
type Client struct {
	address    string
	authToken  string
	httpClient *http.Client
	log        *logrus.Entry

	mu        sync.Mutex
	requestID int64
}

// ClientInterface - request client interface
type ClientInterface interface {
	BuildURI(uri string, params map[string]string) string
	Send(method, path string, data interface{}) (int, []byte, error)
	SetAuthToken(token string)
}

// BuildURI - build request URI using [path?params...] format
func (client *Client) BuildURI(uri string, params map[string]string) string {
	paramsStr := ""
	paramValues := url.Values{}

	for key, val := range params {
		if len(val) != 0 {
			paramValues.Set(key, val)
		}
	}

	paramsStr = paramValues.Encode()
	if len(paramsStr) != 0 {
		uri = fmt.Sprintf("%s?%s", uri, paramsStr)
	}

	return uri
}

// Send - send request to REST server
// data     interface{} - request payload, any interface for json.Marshal()
func (client *Client) Send(method, path string, data interface{}) (int, []byte, error) {
	client.mu.Lock()
	client.requestID++
	l := client.log.WithFields(logrus.Fields{
		"func":  "Send()",
		"req":   fmt.Sprintf("%s %s", method, path),
		"reqID": client.requestID,
	})
	client.mu.Unlock()

	uri := fmt.Sprintf("%s/%s", client.address, path)

	l.Debug("send request")

	// send request data as json
	var jsonDataReader io.Reader
	if data == nil {
		jsonDataReader = nil
	} else {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return 0, nil, err
		}
		jsonDataReader = strings.NewReader(string(jsonData))
		l.Debugf("data: %+v", data) //TODO hide passwords
	}

	req, err := http.NewRequest(method, uri, jsonDataReader)
	if err != nil {
		l.Errorf("request creation error: %s", err)
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if len(client.authToken) != 0 {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", client.authToken))
	}

	res, err := client.httpClient.Do(req)
	if err != nil {
		l.Errorf("request error: %s", err)
		return 0, nil, err
	}

	defer res.Body.Close()

	l.Debugf("response status code: %d", res.StatusCode)

	// validate response body
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		err = fmt.Errorf("Cannot read body of request '%s %s': '%s'", method, uri, err)
		return res.StatusCode, nil, err
	}

	return res.StatusCode, bodyBytes, err
}

// SetAuthToken - set Bearer auth token for all requests
func (client *Client) SetAuthToken(token string) {
	client.authToken = token
}

// ClientArgs - params to create Client instance
type ClientArgs struct {
	Address string
	Log     *logrus.Entry
}

// NewClient - create new REST client
func NewClient(args ClientArgs) (client ClientInterface, err error) {
	l := args.Log.WithField("cmp", "RestClient")

	l.Debugf("created for %s", args.Address)

	tr := &http.Transport{
		IdleConnTimeout: 60 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // don't check certificate, fix this!
		},
	}

	httpClient := &http.Client{
		Transport: tr,
		Timeout:   requestTimeout,
	}

	client = &Client{
		address:    args.Address,
		httpClient: httpClient,
		log:        l,
		requestID:  0,
	}

	return client, nil
}

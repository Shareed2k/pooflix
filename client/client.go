package client

import (
	"bytes"
	"encoding/json"
	"github.com/pooflix/engine"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
)

type Client struct {
	BaseURL    *url.URL
	UserAgent  string
	httpClient *http.Client
}

func NewClient(bu *url.URL) *Client {
	return &Client{
		BaseURL:    bu,
		httpClient: http.DefaultClient,
		UserAgent:  "PooFlix Client",
	}
}

func (c *Client) ListTorrents() (map[string]*engine.Torrent, error) {
	req, err := c.newRequest("GET", "/torrents", nil)
	if err != nil {
		return nil, err
	}

	torrents := make(map[string]*engine.Torrent, 0)
	_, err = c.do(req, &torrents)
	return torrents, err
}

func (c *Client) newRequest(method, p string, body interface{}) (*http.Request, error) {
	rel := &url.URL{Path: path.Join(c.BaseURL.Path, p)}
	u := c.BaseURL.ResolveReference(rel)
	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	return req, nil
}

func (c *Client) do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(v)
	if err != nil {
		log.Printf("Client: can't decode, %v", err)
	}
	return resp, nil
}

package backup

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type WebDAVClient struct {
	url      string
	username string
	password string
	http     *http.Client
}

func NewWebDAVClient(url, username, password string) *WebDAVClient {
	return &WebDAVClient{
		url:      strings.TrimSuffix(url, "/"),
		username: username,
		password: password,
		http:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (w *WebDAVClient) authHeader() string {
	cred := w.username + ":" + w.password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(cred))
}

func (w *WebDAVClient) Test(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", w.url+"/", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", w.authHeader())
	req.Header.Set("Depth", "0")
	resp, err := w.http.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("webdav test failed: %d", resp.StatusCode)
	}
	return nil
}

func (w *WebDAVClient) Push(ctx context.Context, path string, data []byte) error {
	url := w.url + path
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", w.authHeader())
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.http.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webdav push failed: %d", resp.StatusCode)
	}
	return nil
}

func (w *WebDAVClient) Pull(ctx context.Context, path string) ([]byte, error) {
	url := w.url + path
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", w.authHeader())
	resp, err := w.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("webdav pull failed: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

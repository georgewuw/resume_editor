package resume

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

const maxWebDAVPages = 1000

type webDAVStatusError struct {
	status int
}

func (e webDAVStatusError) Error() string { return "WebDAV response status " + strconv.Itoa(e.status) }

type webDAVClient struct {
	baseURL  *url.URL
	username string
	password string
	http     *http.Client
}

type webDAVEntry struct {
	name string
	etag string
}

type webDAVResourceType struct {
	Collection *struct{} `xml:"collection"`
}

type webDAVProperties struct {
	ResourceType webDAVResourceType `xml:"resourcetype"`
	ETag         string             `xml:"getetag"`
}

type webDAVPropstat struct {
	Status string           `xml:"status"`
	Prop   webDAVProperties `xml:"prop"`
}

type webDAVResponse struct {
	Href     string           `xml:"href"`
	Status   string           `xml:"status"`
	Propstat []webDAVPropstat `xml:"propstat"`
}

func newWebDAVClient(serverURL, username, password string) (*webDAVClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(serverURL))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("invalid WebDAV server URL")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return nil, errors.New("WebDAV server URL must use HTTP or HTTPS")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = ""
	return &webDAVClient{
		baseURL: parsed, username: username, password: password,
		http: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *webDAVClient) target(segments ...string) *url.URL {
	target := *c.baseURL
	target.Path = strings.TrimRight(c.baseURL.Path, "/")
	for _, segment := range segments {
		if segment != "" {
			target.Path += "/" + segment
		}
	}
	target.RawPath = ""
	return &target
}

func (c *webDAVClient) request(method, target string, body []byte, headers http.Header, allowed ...int) ([]byte, http.Header, error) {
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, nil, err
	}
	request, err := http.NewRequest(method, parsed.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	request.SetBasicAuth(c.username, c.password)
	request.Header.Set("User-Agent", "resume-editor")
	for name, values := range headers {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 16<<20))
	if err != nil {
		return nil, nil, err
	}
	for _, status := range allowed {
		if response.StatusCode == status {
			return responseBody, response.Header, nil
		}
	}
	return nil, response.Header, webDAVStatusError{status: response.StatusCode}
}

func (c *webDAVClient) probe(remotePath string) error {
	requestBody := []byte(`<?xml version="1.0" encoding="utf-8"?>` +
		`<d:propfind xmlns:d="DAV:"><d:prop><d:resourcetype/></d:prop></d:propfind>`)
	headers := make(http.Header)
	headers.Set("Depth", "0")
	headers.Set("Content-Type", "application/xml; charset=utf-8")
	_, _, err := c.request("PROPFIND", c.target(strings.Split(remotePath, "/")...).String(), requestBody, headers, 207)
	return err
}

func (c *webDAVClient) readDir(remotePath string) ([]webDAVEntry, error) {
	requestBody := []byte(`<?xml version="1.0" encoding="utf-8"?>` +
		`<d:propfind xmlns:d="DAV:"><d:prop><d:resourcetype/><d:getetag/></d:prop></d:propfind>`)
	headers := make(http.Header)
	headers.Set("Depth", "1")
	headers.Set("Content-Type", "application/xml; charset=utf-8")
	firstURL := c.target(strings.Split(remotePath, "/")...).String()
	pageURL := firstURL
	seenPages := map[string]bool{}
	entries := map[string]webDAVEntry{}
	rootPath := path.Clean(c.target(strings.Split(remotePath, "/")...).Path)
	for pages := 0; pageURL != ""; pages++ {
		if pages >= maxWebDAVPages || seenPages[pageURL] {
			return nil, errors.New("WebDAV directory pagination did not finish")
		}
		seenPages[pageURL] = true
		data, responseHeaders, err := c.request("PROPFIND", pageURL, requestBody, headers, 207)
		if err != nil {
			return nil, err
		}
		var page struct {
			Responses []webDAVResponse `xml:"response"`
		}
		if err := xml.Unmarshal(data, &page); err != nil {
			return nil, fmt.Errorf("invalid WebDAV directory response: %w", err)
		}
		for _, item := range page.Responses {
			if !successfulStatus(item.Status) && !hasSuccessfulPropstat(item.Propstat) {
				continue
			}
			name, ok := webDAVChildName(item.Href, c.baseURL, rootPath)
			if !ok || name == "" || isCollection(item.Propstat) || !isManagedDocumentFilename(name) {
				continue
			}
			etag := ""
			for _, props := range item.Propstat {
				if successfulStatus(props.Status) {
					etag = strings.TrimSpace(props.Prop.ETag)
					break
				}
			}
			entries[name] = webDAVEntry{name: name, etag: etag}
		}
		pageURL, err = nextWebDAVPage(pageURL, responseHeaders)
		if err != nil {
			return nil, err
		}
	}
	result := make([]webDAVEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry)
	}
	return result, nil
}

func successfulStatus(status string) bool {
	parts := strings.Fields(status)
	if len(parts) < 2 {
		return false
	}
	code, err := strconv.Atoi(parts[1])
	return err == nil && code >= 200 && code < 300
}

func hasSuccessfulPropstat(items []webDAVPropstat) bool {
	for _, item := range items {
		if successfulStatus(item.Status) {
			return true
		}
	}
	return false
}

func isCollection(items []webDAVPropstat) bool {
	for _, item := range items {
		if successfulStatus(item.Status) && item.Prop.ResourceType.Collection != nil {
			return true
		}
	}
	return false
}

func webDAVChildName(href string, base *url.URL, rootPath string) (string, bool) {
	parsed, err := url.Parse(href)
	if err != nil {
		return "", false
	}
	if parsed.IsAbs() && (parsed.Scheme != base.Scheme || !strings.EqualFold(parsed.Host, base.Host)) {
		return "", false
	}
	resourcePath := path.Clean(parsed.Path)
	prefix := strings.TrimRight(rootPath, "/") + "/"
	if !strings.HasPrefix(resourcePath, prefix) {
		return "", false
	}
	name := strings.TrimPrefix(resourcePath, prefix)
	return name, name != "" && !strings.Contains(name, "/")
}

func nextWebDAVPage(current string, headers http.Header) (string, error) {
	for _, header := range headers.Values("Link") {
		for _, link := range splitWebDAVLinks(header) {
			end := strings.IndexByte(link, '>')
			if !strings.HasPrefix(link, "<") || end < 0 {
				continue
			}
			params := strings.ToLower(link[end+1:])
			rel := ""
			for _, parameter := range strings.Split(params, ";") {
				key, value, ok := strings.Cut(strings.TrimSpace(parameter), "=")
				if ok && strings.TrimSpace(key) == "rel" {
					rel = strings.Trim(strings.TrimSpace(value), `"'`)
				}
			}
			if !containsWord(rel, "next") {
				continue
			}
			currentURL, err := url.Parse(current)
			if err != nil {
				return "", err
			}
			nextURL, err := currentURL.Parse(link[1:end])
			if err != nil {
				return "", err
			}
			if !strings.EqualFold(nextURL.Host, currentURL.Host) || nextURL.Scheme != currentURL.Scheme || nextURL.User != nil || nextURL.Fragment != "" {
				return "", errors.New("WebDAV pagination link left the server origin")
			}
			return nextURL.String(), nil
		}
	}
	return "", nil
}

func splitWebDAVLinks(value string) []string {
	var links []string
	start := 0
	inAngle, inQuote := false, false
	for index, char := range value {
		switch char {
		case '<':
			inAngle = true
		case '>':
			inAngle = false
		case '"':
			inQuote = !inQuote
		case ',':
			if !inAngle && !inQuote {
				links = append(links, strings.TrimSpace(value[start:index]))
				start = index + 1
			}
		}
	}
	return append(links, strings.TrimSpace(value[start:]))
}

func containsWord(words, target string) bool {
	for _, word := range strings.Fields(words) {
		if word == target {
			return true
		}
	}
	return false
}

func (c *webDAVClient) read(remotePath, filename string) ([]byte, error) {
	data, _, err := c.request("GET", c.target(append(strings.Split(remotePath, "/"), filename)...).String(), nil, nil, http.StatusOK)
	return data, err
}

func (c *webDAVClient) write(remotePath, filename string, data []byte) (string, error) {
	_, headers, err := c.request("PUT", c.target(append(strings.Split(remotePath, "/"), filename)...).String(), data, nil,
		http.StatusOK, http.StatusCreated, http.StatusNoContent)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(headers.Get("ETag")), nil
}

func (c *webDAVClient) remove(remotePath, filename string) error {
	_, _, err := c.request("DELETE", c.target(append(strings.Split(remotePath, "/"), filename)...).String(), nil, nil,
		http.StatusOK, http.StatusAccepted, http.StatusNoContent, http.StatusNotFound)
	return err
}

func (c *webDAVClient) makeDirs(remotePath string) error {
	current := []string{}
	for _, part := range strings.Split(remotePath, "/") {
		if part == "" {
			continue
		}
		current = append(current, part)
		_, _, err := c.request("MKCOL", c.target(current...).String(), nil, nil,
			http.StatusCreated, http.StatusMethodNotAllowed, http.StatusOK)
		if err != nil {
			return err
		}
	}
	return nil
}

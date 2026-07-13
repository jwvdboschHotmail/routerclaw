package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type WebBrowser struct {
	Client *http.Client
}

func NewWebBrowser() *WebBrowser {
	jar, _ := cookiejar.New(nil)
	return &WebBrowser{
		Client: &http.Client{
			Jar:     jar,
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

func (w *WebBrowser) extractPageInfo(html string) string {
	// Truncate html if too big for regex parsing
	if len(html) > 50000 {
		html = html[:50000]
	}

	// Regex to find forms and their inputs
	formRe := regexp.MustCompile(`(?is)<form[^>]*action=["']([^"']+)["'][^>]*>(.*?)</form>`)
	inputRe := regexp.MustCompile(`(?is)<input[^>]*name=["']([^"']+)["'][^>]*>`)

	summary := "=== FORMULARIOS DISPONIBLES ===\n"
	forms := formRe.FindAllStringSubmatch(html, -1)
	if len(forms) == 0 {
		summary += "No se encontraron formularios HTML nativos.\n"
	} else {
		for _, form := range forms {
			action := form[1]
			formContent := form[2]
			inputs := inputRe.FindAllStringSubmatch(formContent, -1)
			inputNames := []string{}
			for _, input := range inputs {
				inputNames = append(inputNames, input[1])
			}
			summary += fmt.Sprintf("- URL Acción: %s | Campos: [%s]\n", action, strings.Join(inputNames, ", "))
		}
	}

	// Remove scripts and styles for clean text extraction
	scriptRe := regexp.MustCompile(`(?is)<script.*?>.*?</script>`)
	styleRe := regexp.MustCompile(`(?is)<style.*?>.*?</style>`)
	html = scriptRe.ReplaceAllString(html, "")
	html = styleRe.ReplaceAllString(html, "")

	// Extract text simply by stripping remaining tags
	tagRe := regexp.MustCompile(`(?is)<[^>]+>`)
	text := tagRe.ReplaceAllString(html, " ")
	
	// Clean whitespace
	spaceRe := regexp.MustCompile(`\s+`)
	text = spaceRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	// Limit text output to avoid context overflow in the LLM
	if len(text) > 1500 {
		text = text[:1500] + "... (truncado por memoria)"
	}

	summary += "\n=== CONTENIDO DE LA PÁGINA ===\n" + text
	return summary
}

func (w *WebBrowser) Visit(targetUrl string) (string, error) {
	if !strings.HasPrefix(targetUrl, "http") {
		targetUrl = "http://" + targetUrl
	}

	req, _ := http.NewRequest("GET", targetUrl, nil)
	// Add generic user agent to bypass simple blocks
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	
	resp, err := w.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read up to 100KB to prevent memory crash on the router
	body, err := io.ReadAll(io.LimitReader(resp.Body, 100*1024))
	if err != nil {
		return "", err
	}

	return w.extractPageInfo(string(body)), nil
}

func (w *WebBrowser) Submit(targetUrl, method string, params map[string]string) (string, error) {
	if !strings.HasPrefix(targetUrl, "http") {
		targetUrl = "http://" + targetUrl
	}
	method = strings.ToUpper(method)
	if method == "" {
		method = "POST"
	}
	
	data := url.Values{}
	for k, v := range params {
		data.Set(k, v)
	}

	var req *http.Request
	if method == "POST" {
		req, _ = http.NewRequest("POST", targetUrl, strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		if len(params) > 0 {
			targetUrl += "?" + data.Encode()
		}
		req, _ = http.NewRequest("GET", targetUrl, nil)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := w.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 100*1024))
	if err != nil {
		return "", err
	}

	return w.extractPageInfo(string(body)), nil
}

func (w *WebBrowser) SearchWeb(query string) (string, error) {
	targetUrl := "https://lite.duckduckgo.com/lite/"
	data := url.Values{}
	data.Set("q", query)

	req, _ := http.NewRequest("POST", targetUrl, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := w.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 100*1024))
	if err != nil {
		return "", err
	}
	html := string(bodyBytes)

	scriptRe := regexp.MustCompile(`(?is)<script.*?>.*?</script>`)
	styleRe := regexp.MustCompile(`(?is)<style.*?>.*?</style>`)
	html = scriptRe.ReplaceAllString(html, "")
	html = styleRe.ReplaceAllString(html, "")

	tagRe := regexp.MustCompile(`(?is)<[^>]+>`)
	text := tagRe.ReplaceAllString(html, " ")
	
	spaceRe := regexp.MustCompile(`\s+`)
	text = spaceRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	if len(text) > 3000 {
		text = text[:3000] + "... (truncado)"
	}

	return fmt.Sprintf("RESULTADOS DE BÚSQUEDA PARA '%s':\n%s", query, text), nil
}

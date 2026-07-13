package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GoogleWorkspace struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	AccessToken  string
	TokenExpiry  time.Time
	Client       *http.Client
}

func NewGoogleWorkspace(clientID, secret, refreshToken string) *GoogleWorkspace {
	return &GoogleWorkspace{
		ClientID:     clientID,
		ClientSecret: secret,
		RefreshToken: refreshToken,
		Client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

func (g *GoogleWorkspace) refreshAccessToken() error {
	if time.Now().Before(g.TokenExpiry) {
		return nil // Still valid
	}

	data := url.Values{}
	data.Set("client_id", g.ClientID)
	data.Set("client_secret", g.ClientSecret)
	data.Set("refresh_token", g.RefreshToken)
	data.Set("grant_type", "refresh_token")

	req, _ := http.NewRequest("POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to refresh token: %s", string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	g.AccessToken = result.AccessToken
	g.TokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-60) * time.Second)
	return nil
}

// CheckUnreadEmails returns a summary of up to 3 unread emails
func (g *GoogleWorkspace) CheckUnreadEmails() (string, error) {
	if err := g.refreshAccessToken(); err != nil {
		return "", err
	}

	reqUrl := "https://gmail.googleapis.com/gmail/v1/users/me/messages?q=is:unread&maxResults=3"
	req, _ := http.NewRequest("GET", reqUrl, nil)
	req.Header.Set("Authorization", "Bearer "+g.AccessToken)

	resp, err := g.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("gmail api error: %s", string(body))
	}

	var result struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
		ResultSizeEstimate int `json:"resultSizeEstimate"`
	}
	json.Unmarshal(body, &result)

	if len(result.Messages) == 0 {
		return "No hay correos nuevos.", nil
	}

	summary := fmt.Sprintf("Tienes %d correos sin leer. Aquí están los últimos:\n", result.ResultSizeEstimate)

	for _, msg := range result.Messages {
		msgUrl := fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages/%s?format=metadata&metadataHeaders=Subject&metadataHeaders=From", msg.ID)
		mReq, _ := http.NewRequest("GET", msgUrl, nil)
		mReq.Header.Set("Authorization", "Bearer "+g.AccessToken)
		mResp, err := g.Client.Do(mReq)
		if err != nil {
			continue
		}
		
		mBody, _ := io.ReadAll(mResp.Body)
		mResp.Body.Close()
		
		var mData struct {
			Snippet string `json:"snippet"`
			Payload struct {
				Headers []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"headers"`
			} `json:"payload"`
		}
		json.Unmarshal(mBody, &mData)

		subject := "Sin Asunto"
		from := "Desconocido"
		for _, h := range mData.Payload.Headers {
			if h.Name == "Subject" {
				subject = h.Value
			}
			if h.Name == "From" {
				from = h.Value
			}
		}
		summary += fmt.Sprintf("- ID: %s | De: %s | Asunto: %s | Extracto: %s\n", msg.ID, from, subject, mData.Snippet)
	}

	return summary, nil
}

func (g *GoogleWorkspace) MarkEmailAsRead(id string) error {
	if err := g.refreshAccessToken(); err != nil {
		return err
	}

	reqUrl := fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages/%s/modify", id)
	payload := `{"removeLabelIds": ["UNREAD"]}`
	req, _ := http.NewRequest("POST", reqUrl, strings.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+g.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gmail modify error: %s", string(body))
	}
	return nil
}

func (g *GoogleWorkspace) SendEmail(to, subject, bodyStr string) error {
	if err := g.refreshAccessToken(); err != nil {
		return err
	}

	messageStr := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, bodyStr)
	encoded := base64.URLEncoding.EncodeToString([]byte(messageStr))
	
	payload := fmt.Sprintf(`{"raw": "%s"}`, encoded)
	reqUrl := "https://gmail.googleapis.com/gmail/v1/users/me/messages/send"
	
	req, _ := http.NewRequest("POST", reqUrl, strings.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+g.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gmail send error: %s", string(body))
	}
	return nil
}

func (g *GoogleWorkspace) CreateCalendarEvent(summary, dateTimeStr string) error {
	if err := g.refreshAccessToken(); err != nil {
		return err
	}

	startTime, err := time.Parse(time.RFC3339, dateTimeStr)
	if err != nil {
		if t, err2 := time.Parse("2006-01-02T15:04:05", dateTimeStr); err2 == nil {
			startTime = t
			dateTimeStr = t.Format(time.RFC3339)
			err = nil
		} else if t, err3 := time.Parse("2006-01-02", dateTimeStr); err3 == nil {
			startTime = t
			dateTimeStr = t.Format(time.RFC3339)
			err = nil
		} else {
			return fmt.Errorf("formato de fecha inválido, usa RFC3339: %v", err)
		}
	}
	endTime := startTime.Add(1 * time.Hour).Format(time.RFC3339)

	payload := fmt.Sprintf(`{
		"summary": "%s",
		"start": {"dateTime": "%s"},
		"end": {"dateTime": "%s"}
	}`, summary, dateTimeStr, endTime)

	reqUrl := "https://www.googleapis.com/calendar/v3/calendars/primary/events"
	req, _ := http.NewRequest("POST", reqUrl, strings.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+g.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("calendar create error: %s", string(body))
	}
	return nil
}

// GetTodayEvents returns today's calendar events
func (g *GoogleWorkspace) GetTodayEvents() (string, error) {
	if err := g.refreshAccessToken(); err != nil {
		return "", err
	}

	now := time.Now().Format(time.RFC3339)
	endOfDay := time.Now().AddDate(0, 0, 7).Format(time.RFC3339) // Look ahead 7 days
	
	reqUrl := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/primary/events?timeMin=%s&timeMax=%s&singleEvents=true&orderBy=startTime", url.QueryEscape(now), url.QueryEscape(endOfDay))
	
	req, _ := http.NewRequest("GET", reqUrl, nil)
	req.Header.Set("Authorization", "Bearer "+g.AccessToken)

	resp, err := g.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("calendar api error: %s", string(body))
	}

	var result struct {
		Items []struct {
			Id      string `json:"id"`
			Summary string `json:"summary"`
			Start   struct {
				DateTime string `json:"dateTime"`
			} `json:"start"`
		} `json:"items"`
	}
	json.Unmarshal(body, &result)

	if len(result.Items) == 0 {
		return "No hay eventos programados en los próximos 7 días.", nil
	}

	var events string
	for _, item := range result.Items {
		events += fmt.Sprintf("[%s] %s (ID: %s)\n", item.Start.DateTime, item.Summary, item.Id)
	}
	return events, nil
}

// DeleteCalendarEvent deletes a calendar event by its ID
func (g *GoogleWorkspace) DeleteCalendarEvent(eventId string) error {
	if err := g.refreshAccessToken(); err != nil {
		return err
	}

	reqUrl := "https://www.googleapis.com/calendar/v3/calendars/primary/events/" + url.PathEscape(eventId)
	req, _ := http.NewRequest("DELETE", reqUrl, nil)
	req.Header.Set("Authorization", "Bearer "+g.AccessToken)

	resp, err := g.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("calendar delete error: %s", string(body))
	}
	return nil
}

type EmailData struct {
	ID      string
	From    string
	Subject string
	Snippet string
}

func (g *GoogleWorkspace) GetRawUnreadEmails() ([]EmailData, error) {
	if err := g.refreshAccessToken(); err != nil {
		return nil, err
	}
	reqUrl := "https://gmail.googleapis.com/gmail/v1/users/me/messages?q=is:unread&maxResults=3"
	req, _ := http.NewRequest("GET", reqUrl, nil)
	req.Header.Set("Authorization", "Bearer "+g.AccessToken)

	resp, err := g.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("gmail api error")
	}

	var result struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	json.Unmarshal(body, &result)

	var emails []EmailData
	for _, msg := range result.Messages {
		msgUrl := fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages/%s?format=metadata&metadataHeaders=Subject&metadataHeaders=From", msg.ID)
		mReq, _ := http.NewRequest("GET", msgUrl, nil)
		mReq.Header.Set("Authorization", "Bearer "+g.AccessToken)
		mResp, err := g.Client.Do(mReq)
		if err != nil {
			continue
		}
		
		mBody, _ := io.ReadAll(mResp.Body)
		mResp.Body.Close()
		
		var mData struct {
			Snippet string `json:"snippet"`
			Payload struct {
				Headers []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"headers"`
			} `json:"payload"`
		}
		json.Unmarshal(mBody, &mData)

		subject := "Sin Asunto"
		from := "Desconocido"
		for _, h := range mData.Payload.Headers {
			if h.Name == "Subject" {
				subject = h.Value
			}
			if h.Name == "From" {
				from = h.Value
			}
		}
		emails = append(emails, EmailData{ID: msg.ID, From: from, Subject: subject, Snippet: mData.Snippet})
	}
	return emails, nil
}

type EventData struct {
	ID      string
	Summary string
	Start   string
}

func (g *GoogleWorkspace) GetRawUpcomingEvents() ([]EventData, error) {
	if err := g.refreshAccessToken(); err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)
	endWindow := time.Now().Add(60 * time.Minute).Format(time.RFC3339)
	
	reqUrl := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/primary/events?timeMin=%s&timeMax=%s&singleEvents=true&orderBy=startTime", url.QueryEscape(now), url.QueryEscape(endWindow))
	
	req, _ := http.NewRequest("GET", reqUrl, nil)
	req.Header.Set("Authorization", "Bearer "+g.AccessToken)

	resp, err := g.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("calendar api error")
	}

	var result struct {
		Items []struct {
			ID      string `json:"id"`
			Summary string `json:"summary"`
			Start   struct {
				DateTime string `json:"dateTime"`
			} `json:"start"`
		} `json:"items"`
	}
	json.Unmarshal(body, &result)

	var events []EventData
	for _, item := range result.Items {
		events = append(events, EventData{ID: item.ID, Summary: item.Summary, Start: item.Start.DateTime})
	}
	return events, nil
}

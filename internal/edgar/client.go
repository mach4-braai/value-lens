package edgar

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://data.sec.gov"

type Client struct {
	baseURL   string
	userAgent string
	http      *http.Client
}

func NewClient(baseURL, userAgent string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL:   baseURL,
		userAgent: userAgent,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

type Submissions struct {
	CIK    string `json:"cik"`
	Name   string `json:"name"`
	Filings struct {
		Recent RecentFilings `json:"recent"`
	} `json:"filings"`
}

type RecentFilings struct {
	AccessionNumber []string `json:"accessionNumber"`
	FilingDate      []string `json:"filingDate"`
	Form            []string `json:"form"`
	PrimaryDocument []string `json:"primaryDocument"`
}

type TenKFiling struct {
	AccessionNumber string
	FilingDate      string
	DocumentName    string
}

func (s *Submissions) TenKFilings() []TenKFiling {
	var filings []TenKFiling
	for i, form := range s.Filings.Recent.Form {
		if form == "10-K" {
			filings = append(filings, TenKFiling{
				AccessionNumber: s.Filings.Recent.AccessionNumber[i],
				FilingDate:      s.Filings.Recent.FilingDate[i],
				DocumentName:    s.Filings.Recent.PrimaryDocument[i],
			})
		}
	}
	return filings
}

func (t TenKFiling) DocumentURL(cik string) string {
	noDashes := ""
	for _, c := range t.AccessionNumber {
		if c != '-' {
			noDashes += string(c)
		}
	}
	return fmt.Sprintf(
		"https://www.sec.gov/Archives/edgar/data/%s/%s/%s",
		cik, noDashes, t.DocumentName,
	)
}

func (c *Client) FetchSubmissions(ctx context.Context, cik string) (*Submissions, error) {
	url := fmt.Sprintf("%s/submissions/CIK%s.json", c.baseURL, cik)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch submissions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("EDGAR returned %d", resp.StatusCode)
	}

	var subs Submissions
	if err := json.NewDecoder(resp.Body).Decode(&subs); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &subs, nil
}

func (c *Client) FetchDocument(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("document fetch returned %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

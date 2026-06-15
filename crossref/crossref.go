// Package crossref is the library behind the crossref command line:
// the HTTP client, request shaping, wire decoding, and typed data models
// for the Crossref REST API.
//
// The API is open and requires no authentication key. The polite pool
// (higher rate limits) is accessed by supplying a mailto: address in the
// User-Agent header, which DefaultConfig does automatically.
package crossref

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultBaseURL = "https://api.crossref.org"

// DefaultUserAgent identifies the client and opts into the Crossref polite pool.
const DefaultUserAgent = "crossref-cli/dev (mailto:tamnd87@gmail.com; +https://github.com/tamnd/crossref-cli)"

// ErrNotFound is returned when the API returns a 404 or null body.
var ErrNotFound = errors.New("not found")

// Config holds constructor parameters.
type Config struct {
	BaseURL   string // default: "https://api.crossref.org"
	UserAgent string
	Rate      time.Duration // default: 50ms
	Retries   int           // default: 3
	Timeout   time.Duration // default: 30s
}

// DefaultConfig returns sensible defaults for the Crossref polite pool.
func DefaultConfig() Config {
	return Config{
		BaseURL:   defaultBaseURL,
		UserAgent: DefaultUserAgent,
		Rate:      50 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the Crossref REST API.
type Client struct {
	httpClient *http.Client
	userAgent  string
	baseURL    string
	rate       time.Duration
	retries    int
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client built from cfg.
func NewClient(cfg Config) *Client {
	base := cfg.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		userAgent:  cfg.UserAgent,
		baseURL:    base,
		rate:       cfg.Rate,
		retries:    cfg.Retries,
	}
}

// get fetches a URL with pacing and retries.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, false, ErrNotFound
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// getJSON fetches and JSON-decodes into v. Returns ErrNotFound when the body is null.
func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "null" {
		return ErrNotFound
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

// normaliseDOI strips common DOI prefixes so the value can be URL-encoded cleanly.
func normaliseDOI(doi string) string {
	doi = strings.TrimSpace(doi)
	doi = strings.TrimPrefix(doi, "https://doi.org/")
	doi = strings.TrimPrefix(doi, "http://doi.org/")
	doi = strings.TrimPrefix(doi, "https://dx.doi.org/")
	doi = strings.TrimPrefix(doi, "http://dx.doi.org/")
	doi = strings.TrimPrefix(doi, "doi:")
	return doi
}

// ─── wire types ──────────────────────────────────────────────────────────────

type wireWork struct {
	DOI            string         `json:"DOI"`
	Title          []string       `json:"title"`
	Author         []wireAuthor   `json:"author"`
	ContainerTitle []string       `json:"container-title"`
	Published      *wireDateParts `json:"published"`
	Issued         *wireDateParts `json:"issued"`
	Type           string         `json:"type"`
	Publisher      string         `json:"publisher"`
	ReferencedBy   int            `json:"is-referenced-by-count"`
	URL            string         `json:"URL"`
}

type wireAuthor struct {
	Given  string `json:"given"`
	Family string `json:"family"`
}

type wireDateParts struct {
	DateParts [][]int `json:"date-parts"`
}

type wireJournal struct {
	ISSN      []string      `json:"ISSN"`
	Title     string        `json:"title"`
	Publisher string        `json:"publisher"`
	Subjects  []wireSubject `json:"subjects"`
	URL       string        `json:"URL"`
}

type wireSubject struct {
	Name string `json:"name"`
	ASJC int    `json:"ASJC"`
}

type wireType struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type worksResp struct {
	Message struct {
		Items        []wireWork `json:"items"`
		TotalResults int        `json:"total-results"`
	} `json:"message"`
}

type workResp struct {
	Message wireWork `json:"message"`
}

type journalsListResp struct {
	Message struct {
		Items []wireJournal `json:"items"`
	} `json:"message"`
}

type journalResp struct {
	Message wireJournal `json:"message"`
}

type typesResp struct {
	Message struct {
		Items []wireType `json:"items"`
	} `json:"message"`
}

type wireFunder struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Location string   `json:"location"`
	AltNames []string `json:"alt-names"`
}

type fundersListResp struct {
	Message struct {
		Items []wireFunder `json:"items"`
	} `json:"message"`
}

type wireMember struct {
	ID       int      `json:"id"`
	Name     string   `json:"primary-name"`
	Location string   `json:"location"`
	Prefixes []string `json:"prefixes"`
}

type membersListResp struct {
	Message struct {
		Items []wireMember `json:"items"`
	} `json:"message"`
}

// ─── mapping helpers ─────────────────────────────────────────────────────────

func wireWorkToWork(w wireWork, rank int) Work {
	title := ""
	if len(w.Title) > 0 {
		title = w.Title[0]
	}
	journal := ""
	if len(w.ContainerTitle) > 0 {
		journal = w.ContainerTitle[0]
	}
	year := ""
	if w.Published != nil && len(w.Published.DateParts) > 0 && len(w.Published.DateParts[0]) > 0 {
		year = strconv.Itoa(w.Published.DateParts[0][0])
	} else if w.Issued != nil && len(w.Issued.DateParts) > 0 && len(w.Issued.DateParts[0]) > 0 {
		year = strconv.Itoa(w.Issued.DateParts[0][0])
	}
	doi := strings.ToLower(w.DOI)
	return Work{
		Rank:      rank,
		DOI:       doi,
		Title:     title,
		Authors:   formatAuthors(w.Author),
		Journal:   journal,
		Year:      year,
		Type:      w.Type,
		Citations: w.ReferencedBy,
		URL:       "https://doi.org/" + doi,
	}
}

func formatAuthors(authors []wireAuthor) string {
	const maxShown = 3
	parts := make([]string, 0, len(authors))
	for i, a := range authors {
		if i >= maxShown {
			break
		}
		name := a.Family
		if a.Given != "" {
			runes := []rune(a.Given)
			name = a.Family + " " + string(runes[0])
		}
		parts = append(parts, name)
	}
	s := strings.Join(parts, ", ")
	if len(authors) > maxShown {
		s += ", et al."
	}
	return s
}

func wireJournalToJournal(j wireJournal, rank int) Journal {
	issn := strings.Join(j.ISSN, ", ")
	subjects := make([]string, 0, len(j.Subjects))
	for _, s := range j.Subjects {
		subjects = append(subjects, s.Name)
	}
	u := j.URL
	if u == "" {
		u = "https://www.crossref.org/"
	}
	return Journal{
		Rank:      rank,
		ISSN:      issn,
		Title:     j.Title,
		Publisher: j.Publisher,
		Subjects:  strings.Join(subjects, ", "),
		URL:       u,
	}
}

// ─── public methods ───────────────────────────────────────────────────────────

// SearchWorks searches the /works endpoint by full text. workType may be empty.
func (c *Client) SearchWorks(ctx context.Context, query string, limit int, workType string) ([]Work, error) {
	if limit <= 0 {
		limit = 10
	}
	params := url.Values{}
	params.Set("query", query)
	params.Set("rows", strconv.Itoa(limit))
	params.Set("select", "DOI,title,author,container-title,published,type,publisher,is-referenced-by-count,URL")
	if workType != "" {
		params.Set("filter", "type:"+workType)
	}
	rawURL := c.baseURL + "/works?" + params.Encode()

	var resp worksResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	out := make([]Work, len(resp.Message.Items))
	for i, w := range resp.Message.Items {
		out[i] = wireWorkToWork(w, i+1)
	}
	return out, nil
}

// GetWork fetches a single work by DOI.
func (c *Client) GetWork(ctx context.Context, doi string) (Work, error) {
	doi = normaliseDOI(doi)
	rawURL := c.baseURL + "/works/" + url.PathEscape(doi)

	var resp workResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return Work{}, err
	}
	return wireWorkToWork(resp.Message, 0), nil
}

// GetJournal fetches a single journal by ISSN.
func (c *Client) GetJournal(ctx context.Context, issn string) (Journal, error) {
	rawURL := c.baseURL + "/journals/" + url.PathEscape(issn)

	var resp journalResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return Journal{}, err
	}
	return wireJournalToJournal(resp.Message, 0), nil
}

// SearchJournals searches the /journals endpoint by title keyword.
func (c *Client) SearchJournals(ctx context.Context, query string, limit int) ([]Journal, error) {
	if limit <= 0 {
		limit = 10
	}
	params := url.Values{}
	params.Set("query", query)
	params.Set("rows", strconv.Itoa(limit))
	rawURL := c.baseURL + "/journals?" + params.Encode()

	var resp journalsListResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	out := make([]Journal, len(resp.Message.Items))
	for i, j := range resp.Message.Items {
		out[i] = wireJournalToJournal(j, i+1)
	}
	return out, nil
}

// ListTypes returns all Crossref work type identifiers.
func (c *Client) ListTypes(ctx context.Context) ([]WorkType, error) {
	rawURL := c.baseURL + "/types"

	var resp typesResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	out := make([]WorkType, len(resp.Message.Items))
	for i, t := range resp.Message.Items {
		out[i] = WorkType(t)
	}
	return out, nil
}

func wireFunderToFunder(f wireFunder, rank int) Funder {
	return Funder{
		Rank:     rank,
		ID:       f.ID,
		Name:     f.Name,
		Location: f.Location,
		AltNames: f.AltNames,
		URL:      "https://search.crossref.org/funding?q=" + url.QueryEscape(f.ID),
	}
}

func wireMemberToMember(m wireMember, rank int) Member {
	return Member{
		Rank:     rank,
		ID:       m.ID,
		Name:     m.Name,
		Location: m.Location,
		Prefixes: m.Prefixes,
		URL:      "https://www.crossref.org/members/prep/",
	}
}

// SearchFunders searches the /funders endpoint by name keyword.
func (c *Client) SearchFunders(ctx context.Context, query string, limit int) ([]Funder, error) {
	if limit <= 0 {
		limit = 10
	}
	params := url.Values{}
	if query != "" {
		params.Set("query", query)
	}
	params.Set("rows", strconv.Itoa(limit))
	rawURL := c.baseURL + "/funders?" + params.Encode()

	var resp fundersListResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	out := make([]Funder, len(resp.Message.Items))
	for i, f := range resp.Message.Items {
		out[i] = wireFunderToFunder(f, i+1)
	}
	return out, nil
}

// SearchMembers searches the /members endpoint by name keyword.
func (c *Client) SearchMembers(ctx context.Context, query string, limit int) ([]Member, error) {
	if limit <= 0 {
		limit = 10
	}
	params := url.Values{}
	if query != "" {
		params.Set("query", query)
	}
	params.Set("rows", strconv.Itoa(limit))
	rawURL := c.baseURL + "/members?" + params.Encode()

	var resp membersListResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, err
	}
	out := make([]Member, len(resp.Message.Items))
	for i, m := range resp.Message.Items {
		out[i] = wireMemberToMember(m, i+1)
	}
	return out, nil
}

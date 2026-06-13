package crossref

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(ts *httptest.Server) *Client {
	cfg := DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return NewClient(cfg)
}

func TestGetSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte(`"hello"`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	body, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `"hello"` {
		t.Errorf("body = %q", body)
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`"recovered"`))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := NewClient(cfg)

	start := time.Now()
	body, err := c.get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `"recovered"` {
		t.Errorf("body = %q after retries", body)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

func TestGetNullReturnsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("null"))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	var v any
	err := c.getJSON(context.Background(), srv.URL, &v)
	if err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestSearchWorks(t *testing.T) {
	const body = `{
		"status": "ok",
		"message-type": "work-list",
		"message": {
			"total-results": 2,
			"items": [
				{
					"DOI": "10.1000/xyz1",
					"title": ["First Paper"],
					"author": [{"given": "Alice", "family": "Smith"}, {"given": "Bob", "family": "Jones"}],
					"container-title": ["Nature"],
					"published": {"date-parts": [[2023, 5, 1]]},
					"type": "journal-article",
					"publisher": "Pub1",
					"is-referenced-by-count": 10,
					"URL": "https://doi.org/10.1000/xyz1"
				},
				{
					"DOI": "10.1000/xyz2",
					"title": ["Second Paper"],
					"author": [{"given": "Carol", "family": "White"}],
					"container-title": ["Science"],
					"published": {"date-parts": [[2024, 1, 1]]},
					"type": "journal-article",
					"publisher": "Pub2",
					"is-referenced-by-count": 5,
					"URL": "https://doi.org/10.1000/xyz2"
				}
			]
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	works, err := c.SearchWorks(context.Background(), "test query", 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(works) != 2 {
		t.Fatalf("got %d works, want 2", len(works))
	}
	if works[0].DOI != "10.1000/xyz1" {
		t.Errorf("DOI = %q, want 10.1000/xyz1", works[0].DOI)
	}
	if works[0].Authors != "Smith A, Jones B" {
		t.Errorf("Authors = %q, want 'Smith A, Jones B'", works[0].Authors)
	}
	if works[0].Year != "2023" {
		t.Errorf("Year = %q, want 2023", works[0].Year)
	}
	if works[0].Rank != 1 {
		t.Errorf("Rank = %d, want 1", works[0].Rank)
	}
}

func TestGetWork(t *testing.T) {
	const body = `{
		"status": "ok",
		"message-type": "work",
		"message": {
			"DOI": "10.9999/test",
			"title": ["Test Article"],
			"author": [{"given": "John", "family": "Doe"}],
			"container-title": ["Test Journal"],
			"published": {"date-parts": [[2022, 3, 15]]},
			"type": "journal-article",
			"publisher": "TestPub",
			"is-referenced-by-count": 7,
			"URL": "https://doi.org/10.9999/test"
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	work, err := c.GetWork(context.Background(), "10.9999/test")
	if err != nil {
		t.Fatal(err)
	}
	if work.DOI != "10.9999/test" {
		t.Errorf("DOI = %q, want 10.9999/test", work.DOI)
	}
	if work.Title != "Test Article" {
		t.Errorf("Title = %q, want Test Article", work.Title)
	}
	if work.Year != "2022" {
		t.Errorf("Year = %q, want 2022", work.Year)
	}
}

func TestGetJournal(t *testing.T) {
	const body = `{
		"status": "ok",
		"message-type": "journal",
		"message": {
			"ISSN": ["0028-0836", "1476-4687"],
			"title": "Nature",
			"publisher": "Springer Nature",
			"subjects": [{"name": "Multidisciplinary", "ASJC": 1000}],
			"URL": "https://www.nature.com"
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	journal, err := c.GetJournal(context.Background(), "0028-0836")
	if err != nil {
		t.Fatal(err)
	}
	if journal.Title != "Nature" {
		t.Errorf("Title = %q, want Nature", journal.Title)
	}
	if journal.ISSN != "0028-0836, 1476-4687" {
		t.Errorf("ISSN = %q, want '0028-0836, 1476-4687'", journal.ISSN)
	}
}

func TestListTypes(t *testing.T) {
	const body = `{
		"status": "ok",
		"message-type": "type-list",
		"message": {
			"total-results": 3,
			"items": [
				{"id": "journal-article", "label": "Journal Article"},
				{"id": "book-chapter", "label": "Book Chapter"},
				{"id": "dataset", "label": "Dataset"}
			]
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	types, err := c.ListTypes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 3 {
		t.Fatalf("got %d types, want 3", len(types))
	}
	if types[0].ID != "journal-article" {
		t.Errorf("ID = %q, want journal-article", types[0].ID)
	}
	if types[2].ID != "dataset" {
		t.Errorf("ID = %q, want dataset", types[2].ID)
	}
}

func TestFormatAuthors(t *testing.T) {
	tests := []struct {
		name    string
		authors []wireAuthor
		want    string
	}{
		{
			name:    "one author",
			authors: []wireAuthor{{Given: "Alice", Family: "Smith"}},
			want:    "Smith A",
		},
		{
			name: "three authors",
			authors: []wireAuthor{
				{Given: "Alice", Family: "Smith"},
				{Given: "Bob", Family: "Jones"},
				{Given: "Carol", Family: "White"},
			},
			want: "Smith A, Jones B, White C",
		},
		{
			name: "five authors truncated",
			authors: []wireAuthor{
				{Given: "Alice", Family: "Smith"},
				{Given: "Bob", Family: "Jones"},
				{Given: "Carol", Family: "White"},
				{Given: "Dave", Family: "Brown"},
				{Given: "Eve", Family: "Green"},
			},
			want: "Smith A, Jones B, White C, et al.",
		},
		{
			name:    "no given name",
			authors: []wireAuthor{{Family: "Collaboration"}},
			want:    "Collaboration",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatAuthors(tc.authors)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

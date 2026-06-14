package crossref

// Work is the record emitted for a scholarly work.
type Work struct {
	Rank      int    `json:"rank"`
	DOI       string `json:"doi"`
	Title     string `json:"title"`
	Authors   string `json:"authors"`
	Journal   string `json:"journal"`
	Year      string `json:"year"`
	Type      string `json:"type"`
	Citations int    `json:"citations"`
	URL       string `json:"url"`
}

// Journal is the record emitted for a serial journal.
type Journal struct {
	Rank      int    `json:"rank"`
	ISSN      string `json:"issn"`
	Title     string `json:"title"`
	Publisher string `json:"publisher"`
	Subjects  string `json:"subjects"`
	URL       string `json:"url"`
}

// WorkType is the record emitted for a Crossref work type.
type WorkType struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Funder is the record emitted for a research funder registered with Crossref.
type Funder struct {
	Rank     int      `json:"rank"`
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Location string   `json:"location"`
	AltNames []string `json:"alt_names"`
	URL      string   `json:"url"`
}

// Member is the record emitted for a publisher member of Crossref.
type Member struct {
	Rank     int      `json:"rank"`
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Location string   `json:"location"`
	Prefixes []string `json:"prefixes"`
	URL      string   `json:"url"`
}

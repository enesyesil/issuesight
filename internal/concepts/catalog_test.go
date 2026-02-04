package concepts

import "testing"

func TestCatalogMatchKeywords(t *testing.T) {
	catalog, err := LoadCatalog()
	if err != nil {
		t.Fatalf("LoadCatalog error: %v", err)
	}

	entries := catalog.MatchKeywords([]string{"Iceberg", "Data Lake"})
	if !containsSlug(entries, "apache-iceberg") {
		t.Fatalf("expected apache-iceberg in results")
	}
	if !containsSlug(entries, "data-lake") {
		t.Fatalf("expected data-lake in results")
	}
	if !containsSlug(catalog.Entries, "contributing-to-open-source-101") {
		t.Fatalf("expected contributing-to-open-source-101 in catalog")
	}
}

func TestCatalogMatchSignals(t *testing.T) {
	catalog, err := LoadCatalog()
	if err != nil {
		t.Fatalf("LoadCatalog error: %v", err)
	}

	signals := Signals{
		Language: "Java",
		Title:    "Integrate Apache Iceberg table cleanup",
		Labels:   []string{"data lake", "spark"},
	}
	entries := catalog.MatchSignals(signals)

	if !containsSlug(entries, "java") {
		t.Fatalf("expected java in results")
	}
	if !containsSlug(entries, "apache-iceberg") {
		t.Fatalf("expected apache-iceberg in results")
	}
	if !containsSlug(entries, "data-lake") {
		t.Fatalf("expected data-lake in results")
	}
}

func containsSlug(entries []Entry, slug string) bool {
	for _, entry := range entries {
		if entry.Slug == slug {
			return true
		}
	}
	return false
}

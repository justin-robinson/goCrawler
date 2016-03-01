package crawler

import (
	"testing"
)

func TestCrawl(t *testing.T) {

	urlsExpected := map[string]bool{
		"https://jor.pw":                                true,
		"https://jor.pw/downloads/robinson,justin.docx": true,
		"https://jor.pw/downloads/robinson,justin.pdf":  true,
		"https://github.com/justin-robinson":            true,
	}

	urlsGot := map[string]bool{}

	ch := make(chan CrawledUrlResponse)
	quit := make(chan int)

	crawler := Crawler{
		BaseUrl: "https://jor.pw",
		Depth:   2,
		Export:  ch,
		Quit:    quit,
	}

	go crawler.Crawl()

	for {
		select {
		case url := <-crawler.Export:
			urlsGot[url.Url] = true
		case <-quit:

			// check we got the right number of urls
			if len(urlsGot) != len(urlsExpected) {
				t.Errorf("Expected %v urls, got %v", len(urlsExpected), len(urlsGot))
			}

			// check that we didn't get a url we weren't expecting
			for url := range urlsGot {
				if !urlsExpected[url] {
					t.Errorf("Got unexpected url: %v", url)
				} else {
					delete(urlsExpected, url)
				}
			}

			// are we missing a url?
			for url := range urlsExpected {
				if !urlsGot[url] {
					t.Errorf("Expected url: %v", url)
				}
			}
			return
		}
	}

}

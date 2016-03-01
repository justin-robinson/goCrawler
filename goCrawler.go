package crawler

import (
	"log"
	"net/http"
	"net/url"
	"github.com/golang/net/html"
	"io/ioutil"
	"strings"
	"sync"
)

type Crawler struct {
	BaseUrl string
	Depth   int
	Export chan CrawledUrlResponse
	Quit chan int
	mux sync.Mutex
	crawledUrls map[string]bool
}

type CrawledUrlResponse struct {
	Url string
	Body string
}

var startChannel = make(chan bool)
var endChannel = make(chan bool)

var numStarted, numFinished = 1, 0

func (c *Crawler) Crawl() {

	c.crawledUrls = make(map[string]bool)

	go c.crawl(c.BaseUrl, c.Depth)

	for {
		select{
		case <- startChannel:
			numStarted++

		case <- endChannel:
			numFinished++
			if numStarted == numFinished {
				c.Quit <- 1
			}
		}
	}
}

func (c *Crawler) crawl(urlString string, depth int) {

	// max depth?
	if depth <= 0 {
		endChannel <- true
		return
	}

	// have we crawled this url before?
	c.mux.Lock()
	_, crawled := c.crawledUrls[urlString]
	c.mux.Unlock()

	if crawled {
		endChannel <- true
		return
	}

	mainUrl, err := url.Parse(urlString)
	if err != nil {
		log.Print(err)
		endChannel <- true
		return
	}

	// only crawl http or https pages
	switch mainUrl.Scheme {
	case "http":
		fallthrough
	case "https":
	default:
		endChannel <- true
		return
	}

	// mark that we have crawled this
	c.mux.Lock()
	c.crawledUrls[urlString] = true
	c.mux.Unlock()

	// crawl the page
	response, err := http.Get(urlString)
	if err != nil {
		log.Print(err)
		endChannel <- true
		return
	}

	// read body into string
	body, err := ioutil.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		log.Print(err)
		endChannel <- true
		return
	}

	// create our response
	crawledUrl := CrawledUrlResponse{
		urlString,
		string(body),
	}
	// export the response
	c.Export <- crawledUrl

	// parse body
	doc, err := html.Parse(strings.NewReader(crawledUrl.Body))
	if err != nil {
		log.Print(err)
		endChannel <- true
		return
	}

	// find all hrefs
	urls := crawlNodes(doc, mainUrl)


	for _, urlToCrawl := range urls {
		startChannel <- true
		// crawl all links we found
		go c.crawl(urlToCrawl, depth - 1)
	}

	endChannel <- true
	return

}

func crawlNodes(n *html.Node, mainUrl *url.URL) []string {

	urls := []string{}

	if n.Type == html.ElementNode && n.Data == "a" {
		for _, a := range n.Attr {
			if a.Key == "href" {
				u, err := url.Parse(a.Val)
				if err != nil {
					break
				}

				if u.Scheme == "" {
					u.Scheme = mainUrl.Scheme
				}

				if u.Host == "" {
					u.Host = mainUrl.Host
				}

				urls = append(urls, u.String())
				break
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		urls = append(urls, crawlNodes(c, mainUrl)...)
	}

	return urls
}

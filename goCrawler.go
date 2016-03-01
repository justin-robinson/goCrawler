/**
PACKAGE DOCUMENTATION

package crawler
    import "."


TYPES

type CrawledUrlResponse struct {
    Url  string
    Body string
}
    sent back to user after each successful page crawl

type Crawler struct {
    BaseUrl string
    Depth   int
    Export  chan CrawledUrlResponse
    Quit    chan int
    // contains filtered or unexported fields
}
    main struct

func (c *Crawler) Crawl()
    Kicks off the crawling of the baseUrl and keeps track of running
    goroutines
*/
package crawler

import (
	"golang.org/x/net/html"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// main struct
type Crawler struct {
	BaseUrl     string
	Depth       int
	Export      chan CrawledUrlResponse
	Quit        chan int
	mux         sync.Mutex
	crawledUrls map[string]bool
}

// sent back to user after each successful page crawl
type CrawledUrlResponse struct {
	Url  string
	Body string
}

// channels used when crawls are starting and finishing
var startChannel = make(chan bool)
var endChannel = make(chan bool)

// count of how many crawls are running
var numStarted, numFinished = 1, 0

// Kicks off the crawling of the baseUrl and
// keeps track of running goroutines
func (c *Crawler) Crawl() {

	c.crawledUrls = make(map[string]bool)

	go c.crawl(c.BaseUrl, c.Depth)

	for {
		select {
		case <-startChannel:
			numStarted++

		case <-endChannel:
			numFinished++
			if numStarted == numFinished {
				c.Quit <- 1
			}
		}
	}
}

// goroutine function that crawls a specific url and
// kicks off seperate goroutines of found urls in the page
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

	// use this to build relative hrefs that don't
	// have a domain name attached
	originUrl, err := url.Parse(urlString)
	if err != nil {
		log.Print(err)
		endChannel <- true
		return
	}

	// only crawl http or https pages
	switch originUrl.Scheme {
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
	headNode, err := html.Parse(strings.NewReader(crawledUrl.Body))
	if err != nil {
		log.Print(err)
		endChannel <- true
		return
	}

	// find all urls
	urls := findUrlsInNode(headNode, originUrl)

	for _, urlToCrawl := range urls {
		startChannel <- true
		// crawl all links we found
		go c.crawl(urlToCrawl, depth-1)
	}

	endChannel <- true
	return

}

// recursively crawls dom nodes searching for more urls
func findUrlsInNode(n *html.Node, originUrl *url.URL) []string {

	// the url we find
	urls := []string{}

	if n.Type == html.ElementNode && n.Data == "a" {
		for _, a := range n.Attr {
			if a.Key == "href" {
				u, err := url.Parse(a.Val)
				if err != nil {
					break
				}

				// ensure url has a scheme and host
				if u.Scheme == "" {
					u.Scheme = originUrl.Scheme
				}
				if u.Host == "" {
					u.Host = originUrl.Host
				}

				urls = append(urls, u.String())
				break
			}
		}
	}

	// search all sibling nodes
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		urls = append(urls, findUrlsInNode(c, originUrl)...)
	}

	return urls
}

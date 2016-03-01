# goCrawler

A parallel web crawler

## Note:
You need to have `https://github.com/golang/net/tree/master/html`
as `golang.org/x/net/html` in your GOPATH
```shell
git clone git@github.com:golang/net.git $GOPATH/src/golang.org/x/net
```

```go
    // you need 2 channels for data and stopping
    ch := make(chan CrawledUrlResponse)
	quit := make(chan int)

	crawler := Crawler{
		BaseUrl:"https://jor.pw", // where we are starting
		Depth:2,                  // how deep we will follow links
		Export:ch,                // the channel used when a paged is scraped
		Quit: quit,               // the channel telling us we are done
	}

    // start the crawling process
    // This has to be a goroutine
	go crawler.Crawl()

    finished := false
	for !finished {
		select {
		case url := <-crawler.Export:
			// url.Url is the url string
			// url.Body is the content of the page as a string
		case <- quit:
		    // when we get a message from the quit channel we know the
		    // crawl is complete
			finished = true
		}
	}
```

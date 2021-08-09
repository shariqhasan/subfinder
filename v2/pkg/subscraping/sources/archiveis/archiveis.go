// Package archiveis is a Archiveis Scraping Engine in Golang
package archiveis

import (
	"context"
	"fmt"
	"io/ioutil"
)

type Source struct{}

func (s *Source) Name() string {
	return "archiveis"
}

func (s *Source) Run(ctx context.Context, domain string, session *subscraping.Session) <-chan subscraping.Result {
	results := make(chan subscraping.Result)

	go func() {
		defer close(results)

		resp, err := session.SimpleGet(ctx, fmt.Sprintf("https://archive.is/*.%s", domain))

		if err != nil {
			results <- subscraping.Result{Source: "archiveis", Type: subscraping.Error, Error: err}
			session.DiscardHTTPResponse(resp)
			return
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			results <- subscraping.Result{Source: "archiveis", Type: subscraping.Error, Error: err}
			resp.Body.Close()
			return
		}

		resp.Body.Close()

		src := string(body)

		for _, subdomain := range session.Extractor.FindAllString(src, -1) {
			results <- subscraping.Result{Source: "archiveis", Type: subscraping.Subdomain, Value: subdomain}
		}
	}()

	return results
}

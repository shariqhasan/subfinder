package dnsrepo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/projectdiscovery/subfinder/v2/pkg/core"
	"github.com/projectdiscovery/subfinder/v2/pkg/session"
	"github.com/projectdiscovery/subfinder/v2/pkg/subscraping"
)

// Source is the passive scraping agent
type Source struct {
	subscraping.BaseSource
}

type DnsRepoResponse []struct {
	Domain string
}

// inits the source before passing to daemon
func (s *Source) Init() {
	s.BaseSource.SourceName = "dnsrepo"
	s.BaseSource.Recursive = false
	s.BaseSource.Default = true
	s.BaseSource.RequiresKey = true
	s.BaseSource.CreateTask = s.dispatcher
}

func (s *Source) dispatcher(domain string) core.Task {
	task := core.Task{
		Domain: domain,
	}
	randomApiKey := s.BaseSource.GetNextKey()

	task.RequestOpts = &session.RequestOpts{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("https://dnsrepo.noc.org/api/?apikey=%s&search=%s", randomApiKey, domain),
		Source: "dnsrepo",
		UID:    randomApiKey,
	}

	task.OnResponse = func(t *core.Task, resp *http.Response, executor *core.Executor) error {
		defer resp.Body.Close()
		responseData, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		resp.Body.Close()
		var result DnsRepoResponse
		err = json.Unmarshal(responseData, &result)
		if err != nil {
			return err
		}
		for _, sub := range result {
			executor.Result <- core.Result{Input: domain,
				Source: "dnsrepo", Type: core.Subdomain, Value: strings.TrimSuffix(sub.Domain, "."),
			}
		}
		return nil
	}
	return task
}

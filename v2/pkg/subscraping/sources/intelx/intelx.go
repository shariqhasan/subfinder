// Package intelx logic
package intelx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	jsoniter "github.com/json-iterator/go"

	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/subfinder/v2/pkg/core"
	"github.com/projectdiscovery/subfinder/v2/pkg/session"
	"github.com/projectdiscovery/subfinder/v2/pkg/subscraping"
)

type searchResponseType struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
}

type selectorType struct {
	Selectvalue string `json:"selectorvalue"`
}

type searchResultType struct {
	Selectors []selectorType `json:"selectors"`
	Status    int            `json:"status"`
}

type requestBody struct {
	Term       string
	Maxresults int
	Media      int
	Target     int
	Terminate  []int
	Timeout    int
}

// Source is the passive scraping agent
type Source struct {
	subscraping.BaseSource
}

// inits the source before passing to daemon
func (s *Source) Init() {
	s.BaseSource.SourceName = "intelx"
	s.BaseSource.Default = true
	s.BaseSource.Recursive = false
	s.BaseSource.RequiresKey = true
	s.BaseSource.CreateTask = s.dispatcher
}

func (s *Source) dispatcher(domain string) core.Task {
	task := core.Task{
		Domain: domain,
	}

	apihost, apikey, _ := subscraping.GetMultiPartKey(s.GetNextKey())

	searchURL := fmt.Sprintf("https://%s/phonebook/search?k=%s", apihost, apikey)
	reqBody := requestBody{
		Term:       domain,
		Maxresults: 100000,
		Media:      0,
		Target:     1,
		Timeout:    20,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		gologger.Debug().Label("intelx").Msg(err.Error())
		return task
	}
	task.RequestOpts = &session.RequestOpts{
		Method:      http.MethodPost,
		URL:         searchURL,
		ContentType: "application/json",
		Body:        bytes.NewBuffer(body),
		Source:      "intelx",
		UID:         apikey,
	}

	task.OnResponse = func(t *core.Task, resp *http.Response, executor *core.Executor) error {
		defer resp.Body.Close()
		var response searchResponseType
		err = jsoniter.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			return err
		}
		if response.ID == "" {
			return nil
		}

		apihost, apikey, _ := subscraping.GetMultiPartKey(s.GetNextKey())
		// fetch responses of seach results
		resultsURL := fmt.Sprintf("https://%s/phonebook/search/result?k=%s&id=%s&limit=10000", apihost, apikey, response.ID)
		tx := core.Task{
			Domain: domain,
		}
		tx.Metdata = 0
		tx.RequestOpts = &session.RequestOpts{
			Method: http.MethodGet,
			URL:    resultsURL,
			Source: "intelx",
			UID:    apikey,
		}

		// Note Has recursion
		tx.OnResponse = func(t *core.Task, resp *http.Response, executor *core.Executor) error {
			var response searchResultType
			err = jsoniter.NewDecoder(resp.Body).Decode(&response)
			if err != nil {
				return err
			}
			// status = response.Status
			for _, hostname := range response.Selectors {
				executor.Result <- core.Result{Input: domain,
					Source: "intelx", Type: core.Subdomain, Value: hostname.Selectvalue,
				}
			}

			// TODO : Incomplete details
			// // check recursively
			// if status == 0 || status == 3{
			// 	rtask := tx.Clone()
			// }
			return nil
		}

		return nil
	}
	return task
}

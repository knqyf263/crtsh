package fetcher

import (
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/parnurzeal/gorequest"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	pb "gopkg.in/cheggaaa/pb.v2"
)

// GenWorkers generate workders
func GenWorkers(num, wait int) chan<- func() {
	tasks := make(chan func())
	for i := 0; i < num; i++ {
		go func() {
			for f := range tasks {
				f()
				time.Sleep(time.Duration(wait) * time.Second)
			}
		}()
	}
	return tasks
}

// FetchURL returns HTTP response body
func FetchURL(url string) (string, error) {
	var errs []error
	httpProxy := viper.GetString("http-proxy")

	resp, body, err := gorequest.New().Proxy(httpProxy).Get(url).Type("text").End()
	if len(errs) > 0 || resp == nil {
		return "", fmt.Errorf("HTTP error. errs: %v, url: %s", err, url)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP error. errs: %v, status code: %d, url: %s", err, resp.StatusCode, url)
	}
	return body, nil
}

// Cert has certificate information
type Cert struct {
	NotBefore              string
	NotAfter               string
	CommonName             string
	OrganizationalUnitName string
	OrganizationName       string
	LocalityName           string
	StateOrProvinceName    string
	CountryName            string
}

// ScrapeURL scrapes URL
func ScrapeURL(url string) (cert *Cert, err error) {
	cert = &Cert{}
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return nil, errors.Wrap(err, "url scarapping failed")
	}
	doc.Find("body > table:nth-child(6) > tbody > tr:nth-child(7) > td").Each(func(_ int, s *goquery.Selection) {
		text := s.Text()
		fields := strings.Fields(text)
		var mode string
		var subject bool
		for i := 0; i < len(fields); i++ {
			field := fields[i]
			if field == "Subject:" {
				subject = true
				continue
			}
			if field == "Not" && fields[i+1] == "Before:" {
				mode = "NotBefore"
				i++
			} else if field == "Not" && fields[i+1] == "After" {
				mode = "NotAfter"
				i += 2
			} else if subject && field == "commonName" {
				mode = "commonName"
				i++
			} else if subject && field == "organizationalUnitName" && fields[i+1] == "=" {
				mode = "organizationalUnitName"
				i++
			} else if subject && field == "organizationName" && fields[i+1] == "=" {
				mode = "organizationName"
				i++
			} else if subject && field == "streetAddress" && fields[i+1] == "=" {
				mode = "streetAddress"
				i++
			} else if subject && field == "localityName" && fields[i+1] == "=" {
				mode = "localityName"
				i++
			} else if subject && field == "stateOrProvinceName" && fields[i+1] == "=" {
				mode = "stateOrProvinceName"
				i++
			} else if subject && field == "countryName" && fields[i+1] == "=" {
				mode = "countryName"
				i++
			} else if subject && field == "serialNumber" && fields[i+1] == "=" {
				mode = "serialNumber"
				i++
			} else if subject && field == "postalCode" && fields[i+1] == "=" {
				mode = "postalCode"
				i++
			} else if field == "Subject" {
				mode = "other"
			} else {
				switch mode {
				case "NotBefore":
					cert.NotBefore += field + " "
				case "NotAfter":
					cert.NotAfter += field + " "
				case "commonName":
					cert.CommonName += field + " "
				case "organizationalUnitName":
					cert.OrganizationalUnitName += field + " "
				case "organizationName":
					cert.OrganizationName += field + " "
				case "localityName":
					cert.LocalityName += field + " "
				case "stateOrProvinceName":
					cert.StateOrProvinceName += field + " "
				case "countryName":
					cert.CountryName += field + " "
				}
			}

		}
	})
	return cert, nil
}

// FetchConcurrently fetches concurrently
func FetchConcurrently(urls []string, concurrency, wait int) (responses []*Cert, err error) {
	reqChan := make(chan string, len(urls))
	resChan := make(chan *Cert, len(urls))
	errChan := make(chan error, len(urls))
	defer close(reqChan)
	defer close(resChan)
	defer close(errChan)

	go func() {
		for _, url := range urls {
			reqChan <- url
		}
	}()

	bar := pb.StartNew(len(urls))
	defer bar.Finish()
	tasks := GenWorkers(concurrency, wait)
	for range urls {
		tasks <- func() {
			select {
			case url := <-reqChan:
				var err error
				for i := 1; i <= 3; i++ {
					var res *Cert

					res, err = ScrapeURL(url)
					if err == nil {
						resChan <- res
						return
					}
					time.Sleep(time.Duration(i*2) * time.Second)
				}
				errChan <- err
			}
		}
		bar.Increment()
	}

	errs := []error{}
	timeout := time.After(10 * 60 * time.Second)
	for range urls {
		select {
		case res := <-resChan:
			responses = append(responses, res)
		case err := <-errChan:
			errs = append(errs, err)
		case <-timeout:
			return nil, fmt.Errorf("Timeout Fetching URL")
		}
	}
	if 0 < len(errs) {
		return nil, fmt.Errorf("%s", errs)

	}
	return responses, nil
}

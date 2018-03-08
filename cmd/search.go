package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/knqyf263/crtsh/fetcher"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "search",
	Long:  `search`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return search()
	},
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().StringP("query", "q", "", "query (e.g. Facebook)")
	searchCmd.Flags().StringP("domain", "d", "", "Domain Name (e.g. %.exmaple.com)")
	searchCmd.Flags().Bool("plain", false, "plain text mode")
	viper.BindPFlag("query", searchCmd.Flags().Lookup("query"))
	viper.BindPFlag("domain", searchCmd.Flags().Lookup("domain"))
	viper.BindPFlag("plain", searchCmd.Flags().Lookup("plain"))
}

// Result : Result
type Result struct {
	IssuerCaID        int    `json:"issuer_ca_id"`
	IssuerName        string `json:"issuer_name"`
	NameValue         string `json:"name_value"`
	MinCertID         int    `json:"min_cert_id"`
	MinEntryTimestamp string `json:"min_entry_timestamp"`
	NotBefore         string `json:"not_before"`
}

func search() (err error) {
	query := viper.GetString("query")
	domain := viper.GetString("domain")
	if query == "" && domain == "" {
		return errors.New("--query or --domain must be specified")
	}

	crtURL := "https://crt.sh/"
	client := &http.Client{Timeout: time.Duration(30) * time.Second}

	values := url.Values{}
	values.Add("output", "json")
	if query != "" {
		values.Add("q", query)
	} else if domain != "" {
		values.Add("q", domain)
	}

	req, err := http.NewRequest("GET", crtURL, nil)
	if err != nil {
		return errors.Wrap(err, "Failed to NewRequest")
	}
	req.URL.RawQuery = values.Encode()
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "Failed to send HTTP request")
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "Failed to ReadAll")
	}

	var urls []string
	var results []Result
	jsons := strings.Split(string(b), "}")
	for _, j := range jsons {
		if len(j) == 0 {
			continue
		}
		j += "}"
		var result Result
		if err = json.Unmarshal([]byte(j), &result); err != nil {
			return errors.Wrap(err, "Failed to unmarshal json")
		}
		results = append(results, result)
	}

	if query != "" {
		for _, result := range results {
			url := fmt.Sprintf("%s?output=json&id=%d", crtURL, result.MinCertID)
			urls = append(urls, url)
		}
		certs, err := fetcher.FetchConcurrently(urls, 5, 0)
		if err != nil {
			return errors.Wrap(err, "Failed to fetch concurrently")
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Common Name", "Organization", "Locality", "Country", "Not After"})
		table.SetColumnColor(tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlackColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlackColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlackColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlackColor},
		)

		for _, cert := range certs {
			if !viper.GetBool("plain") {
				table.Append([]string{cert.CommonName, cert.OrganizationName, cert.LocalityName, cert.CountryName, cert.NotAfter})
			} else {
				fmt.Println(cert.CommonName)
			}
		}
		if !viper.GetBool("plain") {
			table.Render()
		}
	} else if domain != "" {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "Issuer", "Not Before"})
		table.SetColumnColor(tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlackColor},
			tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlackColor},
		)
		for _, result := range results {
			if !viper.GetBool("plain") {
				table.Append([]string{result.NameValue, result.IssuerName, result.NotBefore})
			} else {
				fmt.Println(result.NameValue)
			}
		}
		if !viper.GetBool("plain") {
			table.Render()
		}
	}

	defer resp.Body.Close()
	return nil
}

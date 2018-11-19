package touchngo

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	TransactionTypeUsage  = "Usage"
	TransactionTypeReload = "Reload"
)

type Client struct {
	baseUrl    string
	username   string
	password   string
	httpClient *http.Client
}

type Transaction struct {
	Number         string
	Timestamp      time.Time
	PostedDate     time.Time
	Type           string
	EntryLocation  string
	EntrySP        string
	ExitLocation   string
	ExitSP         string
	ReloadLocation *string
	Amount         string
	Balance        string
	Class          string
	TagNumber      string
}

type GetTransactionsRequest struct {
	From time.Time
	To   time.Time
}

func Last30Days() *GetTransactionsRequest {
	now := time.Now()
	req := &GetTransactionsRequest{
		To: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
	}

	req.From = req.To.Add(-time.Hour * 24 * 30)

	return req
}

func NewClient(baseUrl string, username string, password string) *Client {
	return &Client{
		baseUrl:    baseUrl,
		username:   username,
		password:   password,
		httpClient: http.DefaultClient,
	}
}

func (c *Client) GetTransactions(cardSerialNumber string, request *GetTransactionsRequest) ([]Transaction, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/transactions", c.baseUrl), nil)
	if err != nil {
		return []Transaction{}, err
	}

	req.Header.Set("Authorization",
		"Basic "+base64.StdEncoding.EncodeToString([]byte(c.username+":"+c.password)))

	q := req.URL.Query()
	q.Add("card_serial_number", cardSerialNumber)
	q.Add("from", request.From.Format("02-Jan-2006"))
	q.Add("to", request.To.Format("02-Jan-2006"))

	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return []Transaction{}, err
	}

	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []Transaction{}, err
	}

	var results []struct {
		Number         string  `json:"number"`
		Timestamp      string  `json:"timestamp"`
		PostedDate     string  `json:"posted_date"`
		Type           string  `json:"type"`
		EntryLocation  string  `json:"entry_location"`
		EntrySP        string  `json:"entry_sp"`
		ExitLocation   string  `json:"exit_location"`
		ExitSP         string  `json:"exit_sp"`
		ReloadLocation *string `json:"reload_location"`
		Amount         string  `json:"amount"`
		Balance        string  `json:"balance"`
		Class          string  `json:"class"`
		TagNumber      string  `json:"tag_number"`
	}

	if err := json.Unmarshal(b, &results); err != nil {
		return []Transaction{}, err
	}

	transactions := []Transaction{}
	for _, r := range results {
		transactions = append(transactions, Transaction{
			Number:         r.Number,
			Timestamp:      parseTimestamp(r.Timestamp),
			PostedDate:     parseDate(r.PostedDate),
			Type:           r.Type,
			EntryLocation:  r.EntryLocation,
			EntrySP:        r.EntrySP,
			ExitLocation:   r.ExitLocation,
			ExitSP:         r.ExitSP,
			ReloadLocation: r.ReloadLocation,
			Amount:         r.Amount,
			Balance:        r.Balance,
			Class:          r.Class,
			TagNumber:      r.TagNumber,
		})
	}

	return transactions, nil
}

func parseTimestamp(s string) time.Time {
	t, err := time.Parse("02/01/2006 15:04:05", s)
	if err != nil {
		panic(err)
	}

	return t
}

func parseDate(s string) time.Time {
	t, err := time.Parse("02/01/2006", s)
	if err != nil {
		panic(err)
	}

	return t
}

package ynab

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

const ynabBaseURL = "https://api.youneedabudget.com/v1"

type Client struct {
	AccessToken string

	httpClient *http.Client
}

type Account struct {
	ID      string
	Name    string
	Balance string
}

type Transaction struct {
	AccountID  string
	Date       time.Time
	Amount     string
	CategoryID string
	Memo       string
	Cleared    bool
	Approved   bool
	ImportID   string
}

func (t Transaction) toJSON() interface{} {
	amount, err := strconv.ParseFloat(t.Amount, 64)
	if err != nil {
		panic(err)
	}

	cleared := "uncleared"
	if t.Cleared {
		cleared = "cleared"
	}

	return struct {
		AccountID  string `json:"account_id"`
		Date       string `json:"date"`
		Amount     int64  `json:"amount"`
		CategoryID string `json:"category_id,omitempty"`
		Memo       string `json:"memo"`
		Cleared    string `json:"cleared"`
		Approved   bool   `json:"approved"`
		ImportID   string `json:"import_id"`
	}{
		AccountID:  t.AccountID,
		Date:       t.Date.Format("2006-01-02"),
		Amount:     int64(amount * 1000.0),
		CategoryID: t.CategoryID,
		Memo:       t.Memo,
		Cleared:    cleared,
		Approved:   t.Approved,
		ImportID:   t.ImportID,
	}
}

func NewClient(accessToken string, insecure bool) *Client {
	var httpClient *http.Client

	if insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}

		httpClient = &http.Client{
			Transport: tr,
		}
	} else {
		httpClient = http.DefaultClient
	}

	return &Client{
		AccessToken: accessToken,

		httpClient: httpClient,
	}
}

func (c *Client) GetAccount(budgetID string, accountID string) (Account, error) {
	req, err := http.NewRequest(http.MethodGet,
		fmt.Sprintf("%s/budgets/%s/accounts/%s", ynabBaseURL, budgetID, accountID), nil)
	if err != nil {
		return Account{}, err
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Account{}, err
	}

	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Account{}, err
	}

	var result struct {
		Data struct {
			Account struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				Balance int64  `json:"balance"`
			} `json:"account"`
		} `json:"data"`
	}

	if err := json.Unmarshal(b, &result); err != nil {
		return Account{}, err
	}

	account := Account{
		ID:      result.Data.Account.ID,
		Name:    result.Data.Account.Name,
		Balance: strconv.FormatFloat((float64(result.Data.Account.Balance) / float64(1000)), 'f', 2, 64),
	}

	return account, nil
}

func (c *Client) CreateTransactions(budgetID string, transactions []Transaction) error {
	request := struct {
		Transactions []interface{} `json:"transactions"`
	}{
		Transactions: []interface{}{},
	}

	for _, t := range transactions {
		request.Transactions = append(request.Transactions, t.toJSON())
	}

	body, err := json.Marshal(request)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/budgets/%s/transactions", ynabBaseURL, budgetID), bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusBadRequest {
		return errors.New("the request could not be understood due to malformed syntax or validation error(s)")
	} else if resp.StatusCode == http.StatusConflict {
		return errors.New("a transaction on the same account with the same import_id already exists")
	} else if resp.StatusCode != http.StatusCreated {
		return errors.New("unknown error")
	}

	return nil
}

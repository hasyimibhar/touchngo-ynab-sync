package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/hasyimibhar/touchngo-ynab-sync/touchngo"
	"github.com/hasyimibhar/touchngo-ynab-sync/ynab"
)

const ynabBaseURL = "https://api.youneedabudget.com/v1/"

type Account struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Balance int64  `json:"balance"`
}

func main() {
	accessToken := os.Getenv("YNAB_ACCESS_TOKEN")
	budgetID := os.Getenv("YNAB_BUDGET_ID")
	accountID := os.Getenv("YNAB_ACCOUNT_ID")
	categoryID := os.Getenv("YNAB_TOUCHNGO_CATEGORY_ID")
	touchngoURL := os.Getenv("TOUCHNGO_URL")
	username := os.Getenv("TOUCHNGO_USERNAME")
	password := os.Getenv("TOUCHNGO_PASSWORD")
	cardSerialNumber := os.Getenv("TOUCHNGO_CARD_SERIAL_NUMBER")

	insecureStr := os.Getenv("INSECURE")
	insecure := false
	if insecureStr == "true" || insecureStr == "1" {
		insecure = true
		log.Println("WARN: running in insecure mode, which skips TLS verify")
	}

	ynabClient := ynab.NewClient(accessToken, insecure)

	touchngoClient := touchngo.NewClient(touchngoURL, username, password)

	account, err := ynabClient.GetAccount(budgetID, accountID)
	if err != nil {
		log.Printf("ERROR: failed to fetch YNAB account: %s", err)
		os.Exit(1)
	}

	transactions, err := touchngoClient.GetTransactions(cardSerialNumber, touchngo.Last30Days())
	if err != nil {
		log.Printf("ERROR: failed to fetch touchngo transactions: %s", err)
		os.Exit(1)
	}

	newTransactions := []ynab.Transaction{}

	for _, t := range transactions {
		if t.Balance == account.Balance {
			break
		}

		newTransactions = append(newTransactions,
			createYNABTransactionFromTouchngo(accountID, categoryID, t))
	}

	log.Printf("INFO: found %d new transactions\n", len(newTransactions))

	if len(newTransactions) > 0 {
		if err := ynabClient.CreateTransactions(budgetID, newTransactions); err != nil {
			log.Printf("ERROR: failed to import transactions: %s", err)
			os.Exit(1)
		}
	}

	os.Exit(0)
}

func createYNABTransactionFromTouchngo(
	accountID string,
	categoryID string,
	t touchngo.Transaction) ynab.Transaction {

	transaction := ynab.Transaction{
		AccountID: accountID,
		Date:      t.Timestamp,
		Approved:  true,
		Cleared:   false,
		ImportID:  "TOUCHNGO:" + t.Number + ":" + strconv.FormatInt(time.Now().Unix(), 10),
	}

	if t.Type == touchngo.TransactionTypeUsage {
		transaction.CategoryID = categoryID
		transaction.Amount = "-" + t.Amount
		transaction.Memo = fmt.Sprintf("%s - %s", t.EntryLocation, t.ExitLocation)
	} else if t.Type == touchngo.TransactionTypeReload {
		transaction.Amount = t.Amount
		transaction.Memo = "Reload: " + *t.ReloadLocation
	}

	return transaction
}

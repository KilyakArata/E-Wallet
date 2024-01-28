package main

import (
	"bytes"

	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (w *Wallet) TransactionFrom(amount float64) {
	w.Balance -= amount
}

func (w *Wallet) TransactionTo(amount float64) {
	w.Balance += amount
}

type Wallet struct {
	ID      string  `json:"id"`
	Balance float64 `json:"balance"`
}

type PreTransaction struct {
	To     string  `json:"idTo"`
	Amount float64 `json:"amount"`
}

type Transaction struct {
	Time   string  `json:"time"`
	From   string  `json:"idFrom"`
	To     string  `json:"idTo"`
	Amount float64 `json:"amount"`
}

var wallets = map[string]Wallet{
	"d2ceaa81-0cf0-402f-be7e-7e89e0528420": {
		ID:      "d2ceaa81-0cf0-402f-be7e-7e89e0528420",
		Balance: 87.5,
	},
	"295b7ec4-1fab-4ae5-95b4-05c1248bcdb0": {
		ID:      "295b7ec4-1fab-4ae5-95b4-05c1248bcdb0",
		Balance: 34.7,
	},
}

var transactions = map[int]Transaction{
	1: {
		Time:   "2024-01-25T23:40:25+03:00",
		From:   "d2ceaa81-0cf0-402f-be7e-7e89e0528420",
		To:     "295b7ec4-1fab-4ae5-95b4-05c1248bcdb0",
		Amount: 10,
	},
	2: {
		Time:   "2024-01-25T14:35:25+03:00",
		From:   "295b7ec4-1fab-4ae5-95b4-05c1248bcdb0",
		To:     "d2ceaa81-0cf0-402f-be7e-7e89e0528420",
		Amount: 21,
	},
}

func getWallets(w http.ResponseWriter, r *http.Request) {
	resp, err := json.Marshal(wallets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func getTransactions(w http.ResponseWriter, r *http.Request) {
	resp, err := json.Marshal(transactions)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func postWallet(w http.ResponseWriter, r *http.Request) {

	id := uuid.New()

	wallet := Wallet{
		id.String(),
		100.0,
	}

	wallets[id.String()] = wallet

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	io.WriteString(w, "Кошелек добавлен")
}

func getWallet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "walletId")

	wallet, ok := wallets[id]
	if !ok {
		http.Error(w, "Кошелек не найден", http.StatusNotFound)
		return
	}

	resp, err := json.Marshal(wallet)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func postTransaction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "walletId")

	walletFrom, ok := wallets[id]
	if !ok {
		http.Error(w, "Ваш кошелек не найден", http.StatusPaymentRequired)
		return
	}

	_, err := json.Marshal(walletFrom)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var preTransaction PreTransaction
	var buf bytes.Buffer

	_, err = buf.ReadFrom(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = json.Unmarshal(buf.Bytes(), &preTransaction); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	walletTo, ok := wallets[preTransaction.To]
	if !ok {
		http.Error(w, "Кошелек для перевода не найден", http.StatusBadRequest)
		return
	}

	_, err = json.Marshal(walletTo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if preTransaction.Amount > walletFrom.Balance {
		http.Error(w, "Недостаточно денег на счету", http.StatusBadRequest)
		return
	}

	walletFrom.TransactionFrom(preTransaction.Amount)
	walletTo.TransactionTo(preTransaction.Amount)

	wallets[id] = walletFrom
	wallets[preTransaction.To] = walletTo

	var transaction Transaction

	transaction.Time = time.Now().Format(time.RFC3339)
	transaction.From = id
	transaction.To = preTransaction.To
	transaction.Amount = preTransaction.Amount

	transactions[(len(transactions) + 1)] = transaction

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	io.WriteString(w, "Перевод выполнен")
}

func getTransactionsOfWallet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "walletId")

	transactions1 := map[int]Transaction{}
	var j int

	for i := 0; i <= len(transactions); i++ {

		if transactions[i].From == id || transactions[i].To == id {
			j++
			transactions1[j] = transactions[i]
		}
	}

	resp, err := json.Marshal(transactions1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func main() {
	r := chi.NewRouter()

	r.Get("/api/v1/wallet", getWallets)
	r.Post("/api/v1/wallet", postWallet)
	r.Get("/api/v1/wallet/{walletId}", getWallet)
	r.Post("/api/v1/wallet/{walletId}/send", postTransaction)
	r.Get("/api/v1/wallet/history", getTransactions)
	r.Get("/api/v1/wallet/{walletId}/history", getTransactionsOfWallet)

	if err := http.ListenAndServe(":1234", r); err != nil {
		fmt.Printf("Ошибка при запуске сервера: %s", err.Error())
		return
	}
}

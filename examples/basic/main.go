package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/nickmeessen/dag-go/network"
	"github.com/nickmeessen/dag-go/tx"
	"github.com/nickmeessen/dag-go/wallet"
)

// Sample usage of the library, errors are not handled for readability.
func main() {
	ctx := context.Background()

	sender, _ := wallet.FromPrivateKey(os.Getenv("PRIVATE_KEY"))
	recipient, _ := wallet.New()

	client, _ := network.New(network.WithIntegrationNet())

	bal, _ := client.Balance(ctx, sender.Address())
	fmt.Printf("sender balance: %s DAG\n", bal.FormatToken(8))
	fmt.Printf("recipient address: %s\n", recipient.Address())

	ref, _ := client.LastTxRef(ctx, sender.Address())

	testTx := tx.Transfer{
		Source:      sender.Address(),
		Destination: recipient.Address(),
		Amount:      tx.Token(8, 0.001),
		Parent:      ref,
		Fee:         tx.Token(8, 0.0001),
	}

	signedTx, _ := sender.Sign(testTx)

	txHash, err := client.Send(ctx, signedTx)
	if err != nil {
		fmt.Printf("send error: %v\n", err)
		return
	}
	fmt.Printf("submitted: %s\n", txHash)
	for {
		_, err := client.PendingTx(ctx, txHash)
		if errors.Is(err, network.ErrNotFound) {
			break
		}
		time.Sleep(2 * time.Second)
	}
	fmt.Printf("confirmed: %s\n", txHash)
}

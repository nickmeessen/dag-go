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

// Sample usage of the library against a metagraph (currency token), errors
// are not handled for readability. Set MG_L0_URL and MG_L1_URL to the
// metagraph's L0 and L1 endpoints.
func main() {
	ctx := context.Background()

	sender, _ := wallet.FromPrivateKey(os.Getenv("PRIVATE_KEY"))
	recipient, _ := wallet.New()

	client, _ := network.NewMetagraphClient(os.Getenv("MG_L0_URL"), os.Getenv("MG_L1_URL"))

	bal, _ := client.Balance(ctx, sender.Address())
	fmt.Printf("sender balance: %s\n", bal)
	fmt.Printf("recipient address: %s\n", recipient.Address())

	ref, _ := client.LastTxRef(ctx, sender.Address())

	testTx := tx.Transfer{
		Source:      sender.Address(),
		Destination: recipient.Address(),
		Amount:      tx.DAG(0.001),
		Parent:      ref,
		Fee:         tx.DAG(0.0001),
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

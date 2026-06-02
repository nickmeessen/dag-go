package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nickmeessen/dag-go/network"
)

// Sample usage of the library against the Block Explorer, errors are not
// handled for readability. Set ADDRESS to a DAG address with on-chain
// activity, optionally set METAGRAPH_ID to scope reads to a specific
// metagraph's currency token, and TX_HASH to look up a single confirmed
// transaction.
func main() {
	ctx := context.Background()

	addr := os.Getenv("ADDRESS")
	if addr == "" {
		fmt.Println("set ADDRESS env var to a DAG address with on-chain activity")
		os.Exit(1)
	}

	var be network.BlockExplorer
	if mgId := os.Getenv("METAGRAPH_ID"); mgId != "" {
		be, _ = network.NewMetagraphBlockExplorer(mgId, network.WithMainNet())
	} else {
		be, _ = network.NewBlockExplorer(network.WithMainNet())
	}

	bal, _ := be.AddressBalance(ctx, addr)
	fmt.Printf("balance: %s (at ordinal %d)\n", bal.Balance.FormatToken(8), bal.Ordinal)

	txs, next, _ := be.TransactionsByAddress(ctx, addr, network.PageOpts{Limit: 3})
	fmt.Printf("\nlatest %d transactions:\n", len(txs))
	for _, t := range txs {
		fmt.Printf("  %s  %s  %s\n", t.Hash, t.Amount.FormatToken(8), t.Timestamp)
	}

	if next != "" {
		more, _, _ := be.TransactionsByAddress(ctx, addr, network.PageOpts{Limit: 3, Next: next})
		fmt.Printf("\nnext page (%d more):\n", len(more))
		for _, t := range more {
			fmt.Printf("  %s  %s\n", t.Hash, t.Amount.FormatToken(8))
		}
	}

	locks, _, _ := be.TokenLocksByAddress(ctx, addr, network.PageOpts{ActiveOnly: true})
	fmt.Printf("\nactive token locks (%d):\n", len(locks))
	for _, l := range locks {
		fmt.Printf("  %s  %s  parent %s\n", l.Hash, l.Amount.FormatToken(8), l.ParentHash)
	}

	spends, _, _ := be.AllowSpendsByAddress(ctx, addr, network.PageOpts{ActiveOnly: true})
	fmt.Printf("\nactive allow-spends (%d):\n", len(spends))
	for _, sp := range spends {
		fmt.Printf("  %s  %s → %s  valid until epoch %d\n", sp.Hash, sp.Amount.FormatToken(8), sp.Destination, sp.LastValidEpochProgress)
	}

	if hash := os.Getenv("TX_HASH"); hash != "" {
		t, _ := be.Transaction(ctx, hash)
		fmt.Printf("\ntx %s\n  %s → %s\n  amount %s  fee %s\n  in snapshot %d (%s)\n",
			t.Hash, t.Source, t.Destination,
			t.Amount.FormatToken(8), t.Fee.FormatToken(8),
			t.SnapshotOrdinal, t.SnapshotHash)
	}
}

// Package cart sums shopping carts.
package cart

import "fmt"

// Item is one line of the cart.
type Item struct {
	Name  string
	Cents int
	Qty   int
}

// Total is the sum of price times quantity of every item.
func Total(items []Item) int {
	total := 0
	for i := 0; i < len(items); i++ {
		total += items[i].Cents * items[i].Qty
	}
	return total
}

// Describe is a one-line summary of the cart.
func Describe(items []Item) string {
	return fmt.Sprintf("%d itens, total %d centavos", len(items), Total(items))
}

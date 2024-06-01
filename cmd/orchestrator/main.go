package main

import (
	"flag"
	"fmt"
)

func get_bits(n int) int {
	bits := 0
	for n > 0 {
		bits++
		n >>= 1
	}
	return bits
}

func get_masks(bit_count int) (int, int, int) {
	all_mask := 0b1
	last_mask := 0b1
	first_mask := 0b1

	for i := 1; i < bit_count; i++ {
		all_mask = (all_mask << 1) | 1
		last_mask = last_mask >> 1
		first_mask = first_mask << 1
	}

	return all_mask, last_mask, first_mask
}

// Binary tree

type Node struct {
	ID    int
	Left  *Node
	Right *Node
}

func (n *Node) String() string {
	return fmt.Sprintf("%d", n.ID)
}

func main() {
	nodes := flag.Int("n", 16, "Number of nodes to start")
	flag.Parse()

	all_mask, last_mask, first_mask := get_masks(get_bits(*nodes - 1))

	first_tree := Node{ID: 0}
	second_tree := Node{ID: *nodes - 1}

	// for i := 0; i < *nodes; i++ {
	// 	first := (i << 1) & all_mask
	// 	second := first | last_mask
	// 	third := (i >> 1) & all_mask
	// 	fourth := third | first_mask
	// }

	for i := 0; i < *nodes; i++ {
		first := (i << 1) & all_mask
		second := first | last_mask
		third := (i >> 1) & all_mask
		fourth := third | first_mask

	}
}

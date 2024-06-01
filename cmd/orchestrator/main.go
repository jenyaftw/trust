package main

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"

func get_bits(n int) int {
	bits := 0
	for n > 0 {
		bits++
		n >>= 1
	}
	return bits
}

func get_masks(bitCount int) (int, int, int) {
	allMask := 0b1
	lastMask := 0b1
	firstMask := 0b1

	for i := 1; i < bitCount; i++ {
		allMask = (allMask << 1) | 1
		lastMask = lastMask >> 1
		firstMask = firstMask << 1
	}

	return allMask, lastMask, firstMask
}

type Node struct {
	ID    int
	Left  *Node
	Right *Node
}

func (n *Node) String() string {
	return fmt.Sprintf("%d", n.ID)
}

func (n *Node) FillDeBruijn(max int, depth int) {
	bitCount := get_bits(max)
	if depth > bitCount-1 {
		return
	}

	allMask, _, firstMask := get_masks(bitCount)
	first := (n.ID >> 1) & allMask
	second := first | firstMask

	if first == n.ID && second <= max {
		n.Left = &Node{ID: second}
	} else if second == n.ID && first <= max {
		n.Left = &Node{ID: first}
	} else {
		if first <= max {
			n.Left = &Node{ID: first}
		}
		if second <= max {
			n.Right = &Node{ID: second}
			n.Right.FillDeBruijn(max, depth+1)
		}
	}
	n.Left.FillDeBruijn(max, depth+1)
}

func (n *Node) CapturePrint(prefix string, isTail bool, initial bool, builder *strings.Builder) {
	if n.Right != nil {
		newPrefix := prefix
		if isTail {
			newPrefix += "│   "
		} else {
			newPrefix += "    "
		}
		n.Right.CapturePrint(newPrefix, false, false, builder)
	}

	builder.WriteString(prefix)
	if !initial {
		if isTail {
			builder.WriteString("└── ")
		} else {
			builder.WriteString("┌── ")
		}
	} else {
		builder.WriteString("    ")
	}
	color := Green
	if n.ID == 13 || n.ID == 14 || n.ID == 15 {
		color = Red
	}
	if n.ID == 10 || n.ID == 11 || n.ID == 12 {
		color = Yellow
	}
	builder.WriteString(fmt.Sprintf("%s%d%s\n", color, n.ID, Reset))

	if n.Left != nil {
		newPrefix := prefix
		if isTail {
			newPrefix += "    "
		} else {
			newPrefix += "│   "
		}
		n.Left.CapturePrint(newPrefix, true, false, builder)
	}
}

func (n *Node) PrintToString() string {
	var builder strings.Builder
	n.CapturePrint("", true, true, &builder)
	return builder.String()
}

func main() {
	nodes := flag.Int("n", 16, "Number of nodes to start")
	flag.Parse()

	firstTree := Node{ID: 0}
	secondTree := Node{ID: *nodes - 1}

	firstTree.FillDeBruijn(*nodes-1, 0)
	secondTree.FillDeBruijn(*nodes-1, 0)

	for {
		fmt.Print("\033[H\033[2J")
		fmt.Println("Розподілена система захищеного обміну даними")
		fmt.Printf("Загальна кількість вузлів: %d\n\n", *nodes)

		firstLines := strings.Split(firstTree.PrintToString(), "\n")
		secondLines := strings.Split(secondTree.PrintToString(), "\n")

		maxLen := 0
		for _, line := range firstLines {
			if len(line) > maxLen {
				maxLen = len(line)
			}
		}

		for i := 0; i < len(firstLines) || i < len(secondLines); i++ {
			if i < len(firstLines) {
				fmt.Printf("%-*s", maxLen, firstLines[i])
			} else {
				fmt.Printf("%-*s", maxLen, "")
			}
			if i < len(secondLines) {
				fmt.Print(secondLines[i])
			}
			fmt.Println()
		}

		fmt.Print("\nЖивий: ")
		fmt.Printf("%s%d%s • ", Green, 10, Reset)
		fmt.Print("Запускається: ")
		fmt.Printf("%s%d%s • ", Yellow, 3, Reset)
		fmt.Print("Лежить: ")
		fmt.Printf("%s%d%s\n\n", Red, 3, Reset)

		time.Sleep(time.Second * 1)
	}
}

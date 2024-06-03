package main

import (
	"crypto/x509"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/jenyaftw/trust/internal/pkg/crypto"
)

var RSAKeySize = 4096

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

type NetworkNode struct {
	ID     int
	Status int
	IP     string
	Port   int
	Cert   *x509.Certificate
}

var Nodes []*NetworkNode

type Node struct {
	Node  *NetworkNode
	Left  *Node
	Right *Node
}

func (n *Node) String() string {
	return fmt.Sprintf("%d", n.Node.ID)
}

func (n *Node) FillDeBruijn(max int, depth int) {
	bitCount := get_bits(max)
	if depth > bitCount-1 {
		return
	}

	allMask, _, firstMask := get_masks(bitCount)
	first := (n.Node.ID >> 1) & allMask
	second := first | firstMask

	if first == n.Node.ID && second <= max {
		n.Left = &Node{Node: Nodes[second]}
	} else if second == n.Node.ID && first <= max {
		n.Left = &Node{Node: Nodes[first]}
	} else {
		if first <= max {
			n.Left = &Node{Node: Nodes[first]}
		}
		if second <= max {
			n.Right = &Node{Node: Nodes[second]}
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
	color := Red
	if n.Node.Status == 1 {
		color = Yellow
	}
	if n.Node.Status == 2 {
		color = Green
	}
	builder.WriteString(fmt.Sprintf("%s%d%s\n", color, n.Node.ID, Reset))

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

func startNodes() {
	caKey, err := crypto.GenerateRSAKey(RSAKeySize)
	if err != nil {
		fmt.Println(err)
		return
	}

	caCert := crypto.GenerateCACertificate()
	caCertEnc, err := crypto.EncodeCertificate(caCert, caCert, caKey, caKey)
	if err != nil {
		log.Fatal(err)
		return
	}
	caCertStr := base64.StdEncoding.EncodeToString(caCertEnc)

	serial := 8700
	for i := 0; i < len(Nodes); i++ {
		node := Nodes[i]
		node.Status = 1

		key, err := crypto.GenerateRSAKey(RSAKeySize)
		if err != nil {
			log.Fatal(err)
			return
		}
		keyEnc := crypto.EncodeRSAKey(key)
		keyStr := base64.StdEncoding.EncodeToString(keyEnc)

		cert := crypto.GenerateCertificate(int64(serial), 127, 0, 0, 1)
		node.Cert = cert

		certEnc, err := crypto.EncodeCertificate(cert, caCert, key, caKey)
		if err != nil {
			log.Fatal(err)
			return
		}
		certStr := base64.StdEncoding.EncodeToString(certEnc)

		go (func() {
			output, err := exec.Command("go", "run", "cmd/server/main.go", "-cert", certStr, "-key", keyStr, "-ca", caCertStr, "-port", fmt.Sprint(node.Port), "-host", node.IP).Output()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(string(output))
		})()

		serial++
	}
}

func main() {
	nodes := flag.Int("n", 16, "Кількість вузлів")
	flag.Parse()

	for i := 0; i < *nodes; i++ {
		Nodes = append(Nodes, &NetworkNode{
			ID:     i,
			Status: 0,
			IP:     "127.0.0.1",
			Port:   8700 + i,
		})
	}

	startNodes()

	// firstTree := Node{Node: Nodes[0]}
	// secondTree := Node{Node: Nodes[*nodes-1]}

	// firstTree.FillDeBruijn(*nodes-1, 0)
	// secondTree.FillDeBruijn(*nodes-1, 0)

	// for {
	// 	fmt.Print("\033[H\033[2J")
	// 	fmt.Println("Розподілена система захищеного обміну даними")
	// 	fmt.Printf("Загальна кількість вузлів: %d\n\n", *nodes)

	// 	firstLines := strings.Split(firstTree.PrintToString(), "\n")
	// 	secondLines := strings.Split(secondTree.PrintToString(), "\n")

	// 	maxLen := 0
	// 	for _, line := range firstLines {
	// 		if len(line) > maxLen {
	// 			maxLen = len(line)
	// 		}
	// 	}

	// 	for i := 0; i < len(firstLines) || i < len(secondLines); i++ {
	// 		if i < len(firstLines) {
	// 			fmt.Printf("%-*s", maxLen, firstLines[i])
	// 		} else {
	// 			fmt.Printf("%-*s", maxLen, "")
	// 		}
	// 		if i < len(secondLines) {
	// 			fmt.Print(secondLines[i])
	// 		}
	// 		fmt.Println()
	// 	}

	// 	live := 0
	// 	starting := 0
	// 	dead := 0

	// 	for i := 0; i < len(Nodes); i++ {
	// 		node := Nodes[i]
	// 		if node.Status == 0 {
	// 			dead++
	// 		} else if node.Status == 1 {
	// 			starting++
	// 		} else {
	// 			live++
	// 		}
	// 	}

	// 	fmt.Print("\nЖивий: ")
	// 	fmt.Printf("%s%d%s • ", Green, live, Reset)
	// 	fmt.Print("Запускається: ")
	// 	fmt.Printf("%s%d%s • ", Yellow, starting, Reset)
	// 	fmt.Print("Лежить: ")
	// 	fmt.Printf("%s%d%s\n\n", Red, dead, Reset)

	// 	time.Sleep(time.Millisecond * 100)
	// }
}

package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/jenyaftw/trust/internal/pkg/crypto"
)

var RSAKeySize = 4096
var MinPort = 8700
var NodeCount = 16
var Timeout = 5000

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

func (n *Node) FindNode(id int) *Node {
	if n.Node.ID == id {
		return n
	}
	if n.Left != nil {
		if left := n.Left.FindNode(id); left != nil {
			return left
		}
	}
	if n.Right != nil {
		if right := n.Right.FindNode(id); right != nil {
			return right
		}
	}
	return nil
}

func createAndSaveCertificate(id int, caCert *x509.Certificate, caKey *rsa.PrivateKey) error {
	key, err := crypto.GenerateRSAKey(RSAKeySize)
	if err != nil {
		return err
	}
	keyEnc := crypto.EncodeRSAKey(key)

	cert := crypto.GenerateCertificate(int64(id), 127, 0, 0, 1)
	certEnc, err := crypto.EncodeCertificate(cert, caCert, key, caKey)
	if err != nil {
		return err
	}

	ioutil.WriteFile(fmt.Sprintf("certs/client_%d.key", id), keyEnc, 0644)
	ioutil.WriteFile(fmt.Sprintf("certs/client_%d.crt", id), certEnc, 0644)
	return nil
}

func launchNode(id int, serial int, first *Node, second *Node, caCert *x509.Certificate, caKey *rsa.PrivateKey, caCertStr string, timeout int) {
	node := Nodes[id]
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

	peers := make([]string, 0)
	firstNode := first.FindNode(id)
	if firstNode.Left != nil {
		peers = append(peers, fmt.Sprintf("%s:%d", firstNode.Left.Node.IP, firstNode.Left.Node.Port))
	}
	if firstNode.Right != nil {
		peers = append(peers, fmt.Sprintf("%s:%d", firstNode.Right.Node.IP, firstNode.Right.Node.Port))
	}

	secondNode := second.FindNode(id)
	if secondNode.Left != nil {
		peers = append(peers, fmt.Sprintf("%s:%d", secondNode.Left.Node.IP, secondNode.Left.Node.Port))
	}
	if secondNode.Right != nil {
		peers = append(peers, fmt.Sprintf("%s:%d", secondNode.Right.Node.IP, secondNode.Right.Node.Port))
	}

	peersString := strings.Join(peers, ",")

	node.Status = 2
	cmd := exec.Command("./bin/server.exe", "-cert", certStr, "-key", keyStr, "-ca", caCertStr, "-port", fmt.Sprint(node.Port), "-host", node.IP, "-peers", peersString, "-id", fmt.Sprint(id), "-timeout", fmt.Sprint(timeout))
	if err := cmd.Run(); err != nil {
		node.Status = 0
		launchNode(id, serial, first, second, caCert, caKey, caCertStr, timeout)
	}
}

func startNodes(minPort int, first *Node, second *Node, timeout int) {
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

	createAndSaveCertificate(1, caCert, caKey)
	createAndSaveCertificate(2, caCert, caKey)

	serial := minPort
	for i := 0; i < len(Nodes); i++ {
		go launchNode(i, serial, first, second, caCert, caKey, caCertStr, timeout)

		serial++
	}
}

func main() {
	nodes := flag.Int("n", NodeCount, "Кількість вузлів")
	minPort := flag.Int("p", MinPort, "Мінімальний порт")
	timeout := flag.Int("t", Timeout, "Таймаут для під'єднання вузлів (у мс)")
	flag.Parse()

	for i := 0; i < *nodes; i++ {
		Nodes = append(Nodes, &NetworkNode{
			ID:     i,
			Status: 0,
			IP:     "127.0.0.1",
			Port:   *minPort + i,
		})
	}

	firstTree := Node{Node: Nodes[0]}
	secondTree := Node{Node: Nodes[*nodes-1]}

	firstTree.FillDeBruijn(*nodes-1, 0)
	secondTree.FillDeBruijn(*nodes-1, 0)

	go startNodes(*minPort, &firstTree, &secondTree, *timeout)

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

		live := 0
		starting := 0
		dead := 0

		for i := 0; i < len(Nodes); i++ {
			node := Nodes[i]
			if node.Status == 0 {
				dead++
			} else if node.Status == 1 {
				starting++
			} else {
				live++
			}
		}

		fmt.Print("\nЖивий: ")
		fmt.Printf("%s%d%s • ", Green, live, Reset)
		fmt.Print("Запускається: ")
		fmt.Printf("%s%d%s • ", Yellow, starting, Reset)
		fmt.Print("Лежить: ")
		fmt.Printf("%s%d%s\n", Red, dead, Reset)
		fmt.Printf("Порти: %d-%d\n", *minPort, *minPort+*nodes-1)
		fmt.Print("Команди: C - згенерувати клієнтський сертифікат\n")

		time.Sleep(time.Millisecond * 1000)
	}
}

package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jenyaftw/trust/internal/pkg/crypto"
	"github.com/jenyaftw/trust/internal/pkg/structs"
)

var RSAKeySize = 4096
var MinPort = 8700
var NodeCount = 16
var Timeout = 5000

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

func launchNode(id int, serial int, first *structs.TreeNode, second *structs.TreeNode, caCert *x509.Certificate, caKey *rsa.PrivateKey, caCertStr string, timeout int) {
	node := structs.Nodes[id]
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
	cmd := exec.Command("go", "run", "cmd/server/main.go", "-cert", certStr, "-key", keyStr, "-ca", caCertStr, "-port", fmt.Sprint(node.Port), "-host", node.IP, "-peers", peersString, "-id", fmt.Sprint(id), "-timeout", fmt.Sprint(timeout), "-nodes", fmt.Sprint(len(structs.Nodes)))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		node.Status = 0
		launchNode(id, serial, first, second, caCert, caKey, caCertStr, timeout)
	}
}

func startNodes(minPort int, first *structs.TreeNode, second *structs.TreeNode, timeout int) {
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
	for i := 0; i < len(structs.Nodes); i++ {
		go launchNode(i, serial, first, second, caCert, caKey, caCertStr, timeout)

		serial++
	}
}

func main() {
	nodes := flag.Int("n", NodeCount, "Кількість вузлів")
	minPort := flag.Int("p", MinPort, "Мінімальний порт")
	timeout := flag.Int("t", Timeout, "Таймаут для під'єднання вузлів (у мс)")
	debug := flag.Bool("d", false, "Режим дебагу")
	flag.Parse()

	for i := 0; i < *nodes; i++ {
		structs.Nodes = append(structs.Nodes, &structs.NetworkNode{
			ID:     i,
			Status: 0,
			IP:     "127.0.0.1",
			Port:   *minPort + i,
		})
	}

	firstTree := structs.TreeNode{Node: structs.Nodes[0]}
	secondTree := structs.TreeNode{Node: structs.Nodes[*nodes-1]}

	firstTree.FillDeBruijn(*nodes-1, 0)
	secondTree.FillDeBruijn(*nodes-1, 0)

	go startNodes(*minPort, &firstTree, &secondTree, *timeout)

	if *debug {
		for {
		}
	}

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

		for i := 0; i < len(structs.Nodes); i++ {
			node := structs.Nodes[i]
			if node.Status == 0 {
				dead++
			} else if node.Status == 1 {
				starting++
			} else {
				live++
			}
		}

		fmt.Print("\nЖивий: ")
		fmt.Printf("%s%d%s • ", structs.Green, live, structs.Reset)
		fmt.Print("Запускається: ")
		fmt.Printf("%s%d%s • ", structs.Yellow, starting, structs.Reset)
		fmt.Print("Лежить: ")
		fmt.Printf("%s%d%s\n", structs.Red, dead, structs.Reset)
		fmt.Printf("Порти: %d-%d\n", *minPort, *minPort+*nodes-1)
		fmt.Print("Команди: C - згенерувати клієнтський сертифікат\n")

		time.Sleep(time.Millisecond * 1000)
	}
}

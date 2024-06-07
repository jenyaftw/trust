package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jenyaftw/trust/internal/app"
	"github.com/jenyaftw/trust/internal/pkg/flags"
	"github.com/jenyaftw/trust/internal/pkg/structs"
)

func main() {
	flags := flags.ParseClientFlags()

	client, err := app.NewTrustClient(flags)
	if err != nil {
		log.Println(err)
		return
	}

	err = client.Connect(flags.BufferSize)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println("Connected to server")
	for {
		fmt.Print("Select message type (1 - text, 2 - benchmark, 3 - receive text): ")
		var msg int
		_, err := fmt.Scanf("%d\n", &msg)
		if err != nil {
			log.Println(err)
			return
		}

		if msg != 1 && msg != 2 && msg != 3 {
			fmt.Println("Invalid message type")
			continue
		}

		var dest uint64
		if msg == 1 {
			fmt.Print("Enter destination ID: ")
			_, err = fmt.Scanf("%d\n", &dest)
			if err != nil {
				log.Println(err)
				return
			}
		}

		switch msg {
		case 1:
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter text: ")
			text, _ := reader.ReadString('\n')

			err = client.Send([]byte(text), dest)
			if err != nil {
				log.Println(err)
				return
			}
		case 2:
			fmt.Print("Receive = 0, send = 1: ")
			var recv uint64
			_, err = fmt.Scanf("%d\n", &recv)
			if err != nil {
				log.Println(err)
				return
			}

			switch recv {
			case 1:
				fmt.Print("Enter destination ID: ")
				_, err = fmt.Scanf("%d\n", &dest)
				if err != nil {
					log.Println(err)
					return
				}

				bytes := make([]byte, flags.BufferSize-256)
				_, err := rand.Read(bytes)
				if err != nil {
					log.Println(err)
					return
				}

				for {
					err = client.Send(bytes, dest)
					if err != nil {
						log.Println("rip", err)
						return
					}
				}
			case 0:
				read := client.Read()
				bytesReceived := 0
				start := time.Now().UnixMilli()

				fileName := fmt.Sprintf("received_%d.csv", time.Now().Unix())
				file, err := os.Create(fileName)
				if err != nil {
					log.Println(err)
					return
				}
				defer file.Close()

				for {
					data := <-read
					bytesReceived += len(data)
					if time.Now().UnixMilli()-start > 1000 {
						fmt.Printf("%d Bytes per second\n", bytesReceived)
						_, err := file.WriteString(fmt.Sprintf("%d\n", bytesReceived))
						if err != nil {
							log.Println(err)
							return
						}
						start = time.Now().UnixMilli()
						bytesReceived = 0
					}
				}
			}
		case 3:
			read := client.Read()
			for {
				data := <-read
				block, err := structs.DecodeBlock(data)
				if err != nil {
					log.Println(err)
					continue
				}
				fmt.Println(string(block.Data))
			}
		}
	}
}

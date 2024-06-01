package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/jenyaftw/trust/internal/app"
	"github.com/jenyaftw/trust/internal/pkg/flags"
)

func main() {
	flags := flags.ParseClientFlags()

	client, err := app.NewTrustClient(flags)
	if err != nil {
		log.Println(err)
		return
	}

	err = client.Connect()
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println("Connected to server")
	for {
		fmt.Print("Select message type (1 - text): ")
		var msg int
		_, err := fmt.Scanf("%d\n", &msg)
		if err != nil {
			log.Println(err)
			return
		}

		if msg != 1 {
			fmt.Println("Invalid message type")
			continue
		}

		fmt.Print("Enter destination ID: ")
		var dest uint64
		_, err = fmt.Scanf("%d\n", &dest)
		if err != nil {
			log.Println(err)
			return
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
		}
	}
}

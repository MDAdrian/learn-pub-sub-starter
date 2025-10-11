package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	pubsub "github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	route "github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)


func main() {
	fmt.Println("Starting Peril server...")
	connStr := "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(connStr)
	if err != nil {
		log.Fatal("There was an error with rbmq connection: %w", err)
	}
	defer conn.Close()

	fmt.Println("Connection to RBMQ was success!")

	// create channel
	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("There was an error opening the channel: %w", err)
	}
	

	for {
		in := gamelogic.GetInput()
		if len(in) == 0 {
			continue
		}

		firstWord := in[0]
		switch (firstWord) {
		case "pause":
			log.Println("Got Pause")
			pubsub.PublishJSON(ch, route.ExchangePerilDirect, route.PauseKey, route.PlayingState {
				IsPaused: true,	
			})
			break
		case "resume":
			log.Println("Got Resume")
			pubsub.PublishJSON(ch, route.ExchangePerilDirect, route.PauseKey, route.PlayingState {
				IsPaused: false,	
			})
			break
		case "quit":
			log.Println("Got Quit")

			// Clean up resources
			if err := conn.Close(); err != nil {
				log.Printf("error closing connection: %v", err)
			}
			log.Println("goodbye")
			os.Exit(0)
			break
		default:
			log.Println("Unknown command")
		}


	}


	// // wait for ctrl+c
	// signalChan := make(chan os.Signal, 1)
	// signal.Notify(signalChan, os.Interrupt)
	// <-signalChan

	// 	log.Println("shutting down…")

	// // Clean up resources
	// if err := conn.Close(); err != nil {
	// 	log.Printf("error closing connection: %v", err)
	// }

	// log.Println("goodbye")
}

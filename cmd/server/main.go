package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	amqp "github.com/rabbitmq/amqp091-go"
	pubsub "github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	route  "github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
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
	
	pubsub.PublishJSON(ch, route.ExchangePerilDirect, route.PauseKey, route.PlayingState {
		IsPaused: true,	
	})



	// wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan

		log.Println("shutting down…")

	// Clean up resources
	if err := conn.Close(); err != nil {
		log.Printf("error closing connection: %v", err)
	}

	log.Println("goodbye")
}

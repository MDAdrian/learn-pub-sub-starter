package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	game "github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	pubsub "github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	route "github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)


func main() {
	fmt.Println("Starting Peril client...")
	connStr := "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(connStr)
	if err != nil {
		log.Fatal("There was an error with rbmq connection: %w", err)
	}
	defer conn.Close()

	fmt.Println("Connection to RBMQ was success!")

	// ask for username
	username, err := game.ClientWelcome()
	if err != nil {
		log.Fatal("There was an error getting the username: %w", err)
	}

	// bind
	exchange := fmt.Sprintf("%s.%s", route.PauseKey, username)
	pubsub.DeclareAndBind(conn, route.ExchangePerilDirect, exchange, route.PauseKey, pubsub.Transient)

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


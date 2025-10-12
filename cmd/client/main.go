package main

import (
	"fmt"
	"log"
	"os"

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

	publishCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("could not create channel: %v", err)
	}

	// ask for username
	username, err := game.ClientWelcome()
	if err != nil {
		log.Fatal("There was an error getting the username: %w", err)
	}

	// state
	state := game.NewGameState(username)

	pubsub.SubscribeJSON(conn, route.ExchangePerilDirect, 
		fmt.Sprintf("%s.%s", route.PauseKey, username), 
		route.PauseKey,
		pubsub.Transient, 
		handlerPause(state))
	if err != nil {
		log.Fatalf("could not subscribe to army moves: %v", err)
	}
	pubsub.SubscribeJSON(conn, route.ExchangePerilTopic, 
		 fmt.Sprintf("%s.%s", route.ArmyMovesPrefix, username),
		 route.ArmyMovesPrefix+".*", 
		 pubsub.Transient, 
		 handlerMove(state))
	if err != nil {
		log.Fatalf("could not subscribe to pause: %v", err)
	}
	
	// REPL
	for {
		in := game.GetInput()
		if len(in) == 0 {
			continue
		}

		firstWord := in[0]
		switch (firstWord) {
		case "spawn":
			err := state.CommandSpawn(in)
			if err != nil {
				log.Println(err)
			}
		case "move":
			mv, err := state.CommandMove(in)
			if err != nil {
				log.Println(err)
				continue
			} 
			pubsub.PublishJSON(
				publishCh, 
				route.ExchangePerilTopic, 
				fmt.Sprintf("%s.%s", route.ArmyMovesPrefix, mv.Player.Username), 
				mv,
			)
			if err != nil {
				fmt.Printf("error: %s\n", err)
				continue
			}
			fmt.Printf("Moved %v units to %s\n", len(mv.Units), mv.ToLocation)
		case "status":
			state.CommandStatus()
		case "help":
			game.PrintClientHelp()
		case "spam":
			log.Println("Spamming not allowed yet!")
		case "quit":
			game.PrintQuit()
			os.Exit(0)
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

func handlerPause(gs *game.GameState) func(route.PlayingState) {
	return func(ps route.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
	}
}

func handlerMove(gs *game.GameState) func(game.ArmyMove) {
	return func (am game.ArmyMove) {
		defer fmt.Print(">")
		gs.HandleMove(am)
	}
}


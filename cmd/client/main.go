package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

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

	err = pubsub.SubscribeJSON(conn, route.ExchangePerilDirect, 
		fmt.Sprintf("%s.%s", route.PauseKey, username), 
		route.PauseKey,
		pubsub.Transient, 
		handlerPause(state))
	if err != nil {
		log.Fatalf("could not subscribe to army moves: %v", err)
	}
	err = pubsub.SubscribeJSON(conn, route.ExchangePerilTopic, 
		 fmt.Sprintf("%s.%s", route.ArmyMovesPrefix, username),
		 route.ArmyMovesPrefix+".*", 
		 pubsub.Transient, 
		 handlerMove(state, publishCh))
	if err != nil {
		log.Fatalf("could not subscribe to pause: %v", err)
	}
	err = pubsub.SubscribeJSON(
		conn,
		route.ExchangePerilTopic,
		route.WarRecognitionsPrefix,
		route.WarRecognitionsPrefix+".*",
		pubsub.Durable,
		handlerWar(state, publishCh),
	)
	if err != nil {
		log.Fatalf("could not subscribe to war declarations: %v", err)
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
			secondWord := in[1]
			spamCount, err := strconv.Atoi(secondWord)
			if err != nil {
				fmt.Println("conversion error:", err)
				return
			}
			for range spamCount {
				malLog := game.GetMaliciousLog()
				err := publishGameLog(publishCh, username, malLog)
				if err != nil {
					fmt.Printf("error when publishing mal log: %v\n", err)
				}
			}
			log.Printf("Spammed %v messages\n", spamCount)
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

func handlerPause(gs *game.GameState) func(route.PlayingState) pubsub.AckType {
	return func(ps route.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerMove(gs *game.GameState, publishCh *amqp.Channel) func(game.ArmyMove) pubsub.AckType {
	return func(move game.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")

		moveOutcome := gs.HandleMove(move)
		switch moveOutcome {
		case game.MoveOutcomeSamePlayer:
			return pubsub.Ack
		case game.MoveOutComeSafe:
			return pubsub.Ack
		case game.MoveOutcomeMakeWar:
			err := pubsub.PublishJSON(
				publishCh,
				route.ExchangePerilTopic,
				route.WarRecognitionsPrefix+"."+gs.GetUsername(),
				game.RecognitionOfWar{
					Attacker: move.Player,
					Defender: gs.GetPlayerSnap(),
				},
			)
			if err != nil {
				fmt.Printf("error: %s\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		}

		fmt.Println("error: unknown move outcome")
		return pubsub.NackDiscard
	}
}

func handlerWar(gs *game.GameState, publisCh *amqp.Channel) func(dw game.RecognitionOfWar) pubsub.AckType {
	return func(dw game.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")
		warOutcome, winner, loser := gs.HandleWar(dw)
		switch warOutcome {
		case game.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case game.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case game.WarOutcomeOpponentWon:
			err := publishGameLog(
				publisCh,
				gs.GetUsername(),
				fmt.Sprintf("{%s} won a war againts {%s}", winner, loser),
			)
			if err != nil {
				fmt.Printf("error: $s\n", err)
				return pubsub.NackRequeue
			}
			if err != nil {
				fmt.Printf("error: $s\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case game.WarOutcomeYouWon:
			err := publishGameLog(
				publisCh,
				gs.GetUsername(),
				fmt.Sprintf("{%s} won a war againts {%s}", winner, loser),
			)
			if err != nil {
				fmt.Printf("error: $s\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case game.WarOutcomeDraw:
			err := publishGameLog(
				publisCh,
				gs.GetUsername(),
				fmt.Sprintf("A war between {%s} and {%s} resulted in a draw", winner, loser),
			)
			if err != nil {
				fmt.Printf("error: $s\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		}

		fmt.Println("error: unknown war outcome")
		return pubsub.NackDiscard
	}
}

func publishGameLog(ch *amqp.Channel, username, message string) error {
	err := pubsub.PublishGob(ch, 
		string(route.ExchangePerilTopic), 
		string(route.GameLogSlug) + "." + username, 
		route.GameLog {
			CurrentTime: time.Now(),
			Username: username,
			Message: message,
		},
	)
	if err != nil {
		return fmt.Errorf("error when publishing gamelog: %w", err)
	}
	return nil
}


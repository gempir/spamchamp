package main

import (
	"encoding/json"
	"flag"

	"github.com/gempir/bitraft/pkg/config"
	"github.com/gempir/bitraft/pkg/store"
	"github.com/gempir/bitraft/services/speaker/bot"
	log "github.com/sirupsen/logrus"
)

func main() {
	configFile := flag.String("config", "../../config.json", "json config file")
	flag.Parse()

	cfg := config.NewConfig(*configFile)
	rStore := store.NewRedis()
	db := store.NewDatabase(cfg)

	bot := bot.NewBot(cfg, rStore, db)

	go func() {
		topic := rStore.SubscribeSpeakerMessage()
		channel := topic.Channel()
		for msg := range channel {
			var message store.SpeakerMessage

			if err := json.Unmarshal([]byte(msg.Payload), &message); err != nil {
				log.Error(err)
			}

			bot.Say(message.UserID, message.Channel, message.Message)
		}
	}()

	bot.Connect()
}

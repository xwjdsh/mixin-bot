package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/fox-one/mixin-sdk-go"

	bot "github.com/xwjdsh/mixin-bot"
)

var (
	config = flag.String("config", "keystore.json", "keystore file path")
	pin    = flag.String("pin", "", "PIN value")
)

func main() {
	flag.Parse()
	if *pin == "" {
		log.Panicln("pin not set")
	}

	f, err := os.Open(*config)
	if err != nil {
		log.Panicln(err)
	}

	var store mixin.Keystore
	if err := json.NewDecoder(f).Decode(&store); err != nil {
		log.Panicln(err)
	}

	b, err := bot.Init(&store, *pin)
	if err != nil {
		log.Panicln(err)
	}

	b.Run(context.Background())
}

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
)

func main() {
	flag.Parse()

	f, err := os.Open(*config)
	if err != nil {
		log.Panicln(err)
	}

	var store mixin.Keystore
	if err := json.NewDecoder(f).Decode(&store); err != nil {
		log.Panicln(err)
	}

	b, err := bot.Init(&store)
	if err != nil {
		log.Panicln(err)
	}

	b.Run(context.Background())
}

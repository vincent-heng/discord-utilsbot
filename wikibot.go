package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Configuration filled from configuration file
type Configuration struct {
	DiscordBotKey string
	TimeOut       time.Duration
	MaxChars      int
	MaxSentences  int
	WarningChars  int
}

var (
	configuration *Configuration
)

func main() {
	conf, err := loadConfiguration("config.json")
	if err != nil {
		log.Fatal("Can't load config file:", err)
	}
	configuration = &conf

	rand.Seed(time.Now().UTC().UnixNano())

	// Discord
	dg, err := discordgo.New("Bot " + configuration.DiscordBotKey)
	if err != nil {
		log.Fatalf("error creating Discord session: %v", err)
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!wiki ") {
		name := strings.TrimSpace(strings.TrimPrefix(m.Content, "!wiki "))
		log.Printf("Request: %v", name)

		abstract := fetchWikipediaAbstract(name, false)

		if abstract == "No content." {
			log.Println("No content, trying Camel Case")
			abstract = fetchWikipediaAbstract(strings.Title(name), false)
		}
		if abstract == "No content." {
			log.Println("No content, trying fault tolerancy")
			abstract = fetchWikipediaAbstract(name, true)
		}

		log.Printf("Response: %v", abstract)

		// Send
		s.ChannelMessageSend(m.ChannelID, abstract)
	} else if strings.HasPrefix(m.Content, "!wikipic ") {
		name := strings.TrimSpace(strings.TrimPrefix(m.Content, "!wikipic "))
		log.Printf("Request: %v", name)

		pic := fetchWikipediaPicture(name, false)

		if pic == "No content." {
			log.Println("No content, trying Camel Case")
			pic = fetchWikipediaPicture(strings.Title(name), false)
		}
		if pic == "No content." {
			log.Println("No content, trying fault tolerancy")
			pic = fetchWikipediaPicture(name, true)
		}

		log.Printf("Response: %v", pic)

		// Send
		s.ChannelMessageSend(m.ChannelID, pic)

	} else if m.Content == "!wiki" {
		// Random page
		log.Printf("Request a random page")

		abstract := fetchRandomWikipediaAbstract()

		log.Printf("Response: %v", abstract)

		// Send
		s.ChannelMessageSend(m.ChannelID, abstract)
	}
}

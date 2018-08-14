package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/bwmarrin/discordgo"
	"github.com/knakk/sparql"
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

	if strings.HasPrefix(m.Content, "!wiki") {
		name := strings.TrimSpace(strings.TrimPrefix(m.Content, "!wiki"))
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
	}
}

func fetchWikipediaAbstract(query string, faultTolerant bool) string {
	repo, err := sparql.NewRepo("http://fr.dbpedia.org/sparql",
		sparql.DigestAuth("dba", "dba"),
		sparql.Timeout(time.Millisecond*configuration.TimeOut),
	)
	if err != nil {
		log.Fatal(err)
	}

	query = escapeQuery(query)
	log.Printf("Escaped query: %v", query)
	if !hasLetter(query) {
		return "Bad request :unamused:"
	}

	formattedQuery := ""
	if !faultTolerant {
		formattedQuery = `SELECT ?abstract WHERE {
	       ?categorie rdfs:label "` + query + `"@fr .
	       ?categorie dbpedia-owl:abstract ?abstract
	    } LIMIT 1`
	} else {
		formattedQuery = `SELECT ?abstract WHERE {
	       ?categorie rdfs:label ?mylabel .
	       ?mylabel bif:contains "'` + query + `'" .
	       ?categorie dbpedia-owl:abstract ?abstract
	    } LIMIT 1`

	}
	res, err := repo.Query(formattedQuery)
	if err != nil {
		log.Fatal(err)
	}

	str := "No content"
	if len(res.Results.Bindings) > 0 && res.Results.Bindings[0]["abstract"].Value != "" {
		str = res.Results.Bindings[0]["abstract"].Value
	}
	return limitText(str)
}

func limitText(text string) string {
	text = text[:min(configuration.MaxChars, len(text))]

	splittedAbstract := strings.Split(text, ".")

	numberOfSentences := min(configuration.MaxSentences, len(splittedAbstract))

	reducedAbstract := ""
	for i := 0; i < numberOfSentences && len(reducedAbstract) < configuration.WarningChars; i++ {
		reducedAbstract = reducedAbstract + splittedAbstract[i] + "."
	}
	return reducedAbstract
}

func escapeQuery(text string) string {
	// reg, err := regexp.Compile("[^a-zA-Z0-9 ]+")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// return reg.ReplaceAllString(text, "")
	excapedText := strings.Replace(text, "\"", "", -1)
	excapedText = strings.Replace(excapedText, "'", "", -1)
	excapedText = strings.Replace(excapedText, "\\", "", -1)
	excapedText = strings.Replace(excapedText, "*", "", -1)

	return excapedText
}

func hasLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// loadConfiguration loads configuration from json file
func loadConfiguration(configurationFile string) (Configuration, error) {
	configuration := Configuration{}

	file, err := os.Open(configurationFile)
	if err != nil {
		return configuration, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&configuration)
	return configuration, err
}

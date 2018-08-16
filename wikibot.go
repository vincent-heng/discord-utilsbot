package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
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
	} else if m.Content == "!wiki" {
		// Random page
		log.Printf("Request a random page")

		abstract := fetchRandomWikipediaAbstract()

		log.Printf("Response: %v", abstract)

		// Send
		s.ChannelMessageSend(m.ChannelID, abstract)
	}
}

func fetchRandomWikipediaAbstract() string {
	repo, err := sparql.NewRepo("http://fr.dbpedia.org/sparql",
		sparql.DigestAuth("dba", "dba"),
		sparql.Timeout(time.Millisecond*configuration.TimeOut),
	)
	if err != nil {
		log.Println(err)
		return "Server unavailable :confused:"
	}

	//	nbWikipediaPages := 185404515 // SELECT (COUNT(?s) AS ?triples) WHERE { ?s ?p ?o }
	nbWikipediaPages := 10000 // Perf issues
	someRandomNumber := rand.Intn(nbWikipediaPages)

	log.Printf("Random number generated: %v", someRandomNumber)

	formattedQuery := `SELECT distinct ?label ?abstract WHERE {
		?categorie dbpedia-owl:abstract ?abstract .
		?categorie rdfs:label ?label
		filter langMatches(lang(?abstract),'fr')
	}
	ORDER BY ?s OFFSET ` + strconv.Itoa(someRandomNumber) + ` LIMIT 1
	`

	res, err := repo.Query(formattedQuery)
	if err != nil {
		log.Println(err)
		return "Server unavailable :confused:"
	}

	str := "No content"
	if len(res.Results.Bindings) > 0 && res.Results.Bindings[0]["label"].Value != "" && res.Results.Bindings[0]["abstract"].Value != "" {
		str = res.Results.Bindings[0]["label"].Value + " : " + res.Results.Bindings[0]["abstract"].Value
	}
	return limitText(str)
}

func fetchWikipediaAbstract(query string, faultTolerant bool) string {
	repo, err := sparql.NewRepo("http://fr.dbpedia.org/sparql",
		sparql.DigestAuth("dba", "dba"),
		sparql.Timeout(time.Millisecond*configuration.TimeOut),
	)
	if err != nil {
		log.Println(err)
		return "Server unavailable :confused:"
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
				 filter langMatches(lang(?abstract),'fr')
	    } LIMIT 1`
	} else {
		formattedQuery = `SELECT ?abstract WHERE {
	       ?categorie rdfs:label ?mylabel .
	       ?mylabel bif:contains "'` + query + `'" .
	       ?categorie dbpedia-owl:abstract ?abstract
				 filter langMatches(lang(?abstract),'fr')
	    } LIMIT 1`

	}
	res, err := repo.Query(formattedQuery)
	if err != nil {
		log.Println(err)
		return "Server unavailable :confused:"
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
	escapedText := strings.Replace(text, "\"", "", -1)
	escapedText = strings.Replace(escapedText, "'", "", -1)
	escapedText = strings.Replace(escapedText, "\\", "", -1)
	escapedText = strings.Replace(escapedText, "*", "", -1)

	escapedText = escapedText[:min(34+len(escapedText)%2, len(escapedText))]
	return escapedText
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

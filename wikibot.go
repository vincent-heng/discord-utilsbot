package main

import (
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
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	configuration   *Configuration
	dg              *discordgo.Session
	commands        []*discordgo.ApplicationCommand
	commandHandlers = make(map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate))
)

func init() {
	log.Println("Configuration loading...")

	// Configuration
	conf, err := loadConfiguration("config.json")
	if err != nil {
		log.Fatal("Can't load config file:", err)
	}
	configuration = &conf

	// Use "und" for undetermined language
	lang := language.Und
	// Create a Title caser
	titleCaser := cases.Title(lang)

	rand.Seed(time.Now().UTC().UnixNano()) // Deprecated as of 1.20

	// Discord
	dg, err = discordgo.New("Bot " + configuration.DiscordBotKey)
	if err != nil {
		log.Fatalf("error creating Discord session: %v", err)
	}

	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "wiki",
			Description: "Recherche la description Wikipédia",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "recherche",
					Description: "Mot recherché. Laisser vide pour recherche aléatoire",
					Required:    false,
				},
			},
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"wiki": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			search := ""
			if len(options) > 0 {
				search = options[0].StringValue()
			}

			abstract := fetchWikipediaAbstract(search, true)
			if abstract == "No content." {
				log.Println("No content, trying Camel Case")
				abstract = fetchWikipediaAbstract(titleCaser.String(search), false)
			}
			if abstract == "No content." {
				log.Println("No content, trying fault tolerancy")
				abstract = fetchWikipediaAbstract(search, true)
			}

			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: abstract,
				},
			})
		},
	}

	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {
	log.Printf("Starting...")

	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	// Open a websocket connection to Discord and begin listening.
	err := dg.Open()
	if err != nil {
		log.Println("error opening connection,", err)
		return
	}
	dg.ApplicationCommandBulkOverwrite(configuration.ApiID, "", commands)
	defer dg.Close() // Cleanly close down the Discord session after function termination

	// Wait here until CTRL-C or other term signal is received.
	log.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
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
		?categorie dbo:abstract ?abstract .
		?categorie rdfs:label ?label
		filter langMatches(lang(?abstract),'fr')
	}
	OFFSET ` + strconv.Itoa(someRandomNumber) + ` LIMIT 1
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
	if len(query) == 0 {
		return fetchRandomWikipediaAbstract()
	}

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
	       ?categorie dbo:abstract ?abstract
				 filter langMatches(lang(?abstract),'fr')
	    } LIMIT 1`
	} else {
		formattedQuery = `SELECT ?abstract WHERE {
	       ?categorie rdfs:label ?mylabel .
	       ?mylabel bif:contains "'` + query + `'" .
	       ?categorie dbo:abstract ?abstract
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

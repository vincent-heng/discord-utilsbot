package main

import (
	"encoding/json"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/knakk/sparql"
)

func connectDbpedia() (*sparql.Repo, error) {
	return sparql.NewRepo("http://fr.dbpedia.org/sparql",
		sparql.DigestAuth("dba", "dba"),
		sparql.Timeout(time.Millisecond*configuration.TimeOut),
	)
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

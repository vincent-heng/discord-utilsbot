package main

import (
	"log"
	"math/rand"
	"strconv"
)

func fetchWikipediaPicture(query string, faultTolerant bool) string {
	return "Coming soon"
}

func fetchRandomWikipediaAbstract() string {
	repo, err := connectDbpedia()
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
	query = escapeQuery(query)
	log.Printf("Escaped query: %v", query)
	if !hasLetter(query) {
		return "Bad request :unamused:"
	}
	repo, err := connectDbpedia()
	if err != nil {
		log.Println(err)
		return "Server unavailable :confused:"
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

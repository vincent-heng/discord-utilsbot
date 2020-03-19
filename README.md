# Introduction

*Wikibot* is a Discord Bot that fetch the abstract of DBPedia.

# Installation

- Clone this project
```
git clone https://github.com/vincent-heng/discord-utilsbot
```

- Set the Discord API keys in the configuration file
```
cp config-sample.json config.json
vi config.json
```

- Run it with Docker.
```
docker build . -t wikibot:latest
docker run wikibot:latest
```

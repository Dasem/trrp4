package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog/log"
	uuid "github.com/satori/go.uuid"

	"github.com/Dasem/trrp4/model"
)

var botId string

type options struct {
	DispatcherURL string `long:"dispatcher-host" env:"DISPATCHER" required:"true" default:"localhost:8080"`
	Interval      string `long:"interval" env:"INTERVAL" required:"true" default:"10s"`
}

func getId() {
	var res string
	id, err := ioutil.ReadFile("./tmp")
	if err != nil {
		res = uuid.NewV4().String()
		_ = ioutil.WriteFile("./tmp", id, os.FileMode(0777))
	} else {
		uid, err := uuid.FromString(string(id))
		if err != nil {
			res = uuid.NewV4().String()
			_ = ioutil.WriteFile("./tmp", id, os.FileMode(0777))
		} else {
			res = uid.String()
		}
	}
	botId = res
}

func subscribeForCommands(dispatcherHost string) error {
	u := url.URL{Scheme: "wss", Host: dispatcherHost, Path: "/subscribe"}
	log.Info().Msg("Start subscribe")

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	defer c.Close()
	log.Info().Msg("Subscribed")
	cnt := 0
	var museRequest model.DataMuseRequest
	for {
		if err := c.ReadJSON(&museRequest); err != nil {
			return err
		}
		cnt++
		log.Info().Msgf("Got %d dataMuseResult", cnt)

		escaped := url.QueryEscape(museRequest.Word)

		resp, err := http.Get("https://api.datamuse.com/words?max=10&ml=" + escaped)
		if err != nil {
			log.Error().Err(err).Msg("Failed to do request")
			continue
		}

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error().Err(err).Msg("Failed to read data from request")
			continue
		}

		var dataMuseResults model.DataMuseResults
		if err := json.Unmarshal(data, &dataMuseResults); err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal dataMuseResult")
			continue
		}

		if err := c.WriteJSON(dataMuseResults); err != nil {
			return err
		}
		log.Info().Msgf("Sent to router: %#v", dataMuseResults)
	}
}

func main() {
	// var link = "https://www.cbr-xml-daily.ru/daily_json.js"
	var opts options
	if _, err := flags.Parse(&opts); err != nil {
		log.Fatal().Err(err).Msg("Failed to parse opts")
	}
	getId()

	interval, err := time.ParseDuration(opts.Interval)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to parse duration from opts")
	}

	ticker := time.NewTicker(interval)
	errs := make(chan error, 1)
	errs <- subscribeForCommands(opts.DispatcherURL)
	for {
		select {
		case err := <-errs:
			if err != nil {
				log.Error().Err(err).Msg("Failed to subscribe for commands")
			}
		case <-ticker.C:
			log.Info().Msg("Tried to reconnect")
			errs <- subscribeForCommands(opts.DispatcherURL)
		}
	}
}

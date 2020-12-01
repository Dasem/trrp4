package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog/log"

	"github.com/Dasem/trrp4/model"
	"github.com/Dasem/trrp4/pubsub"
)

var (
	connMu sync.Mutex
)

type internalRequest struct {
	ResCh chan model.DataMuseResults
	Req   model.DataMuseRequest
}

type options struct {
	ProjectID     string `long:"projectID" env:"PROJECT_ID" required:"true" default:"refined-byte-297215"`
	DataTopicName string `long:"dataTopicName" env:"DATA_TOPIC_NAME" required:"true" default:"synonyms"`
	DataSubName   string `long:"dataSubName" env:"DATA_SUB_NAME" required:"true" default:"synonyms-sub"`
	Port          string `long:"port" env:"PORT" required:"true" default:"8080"`
}

type service struct {
	dataClient  *pubsub.Client
	resChan     chan model.DataMuseResult
	upgrader    websocket.Upgrader
	connections map[string]chan internalRequest
}

// handle results
func (s *service) HandleResults() {
	for res := range s.resChan {
		// Black magic
		res := res
		log.Info().Msgf("Sent to pubsub from HandleResults: %#v", res)
		data, err := json.Marshal(res)
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal result")
			continue
		}
		// Publish raw bytes to pubsub
		if err := s.dataClient.Publish(context.Background(), data); err != nil {
			log.Error().Err(err).Msg("Failed to publish msg")
			continue
		}
	}
}

// Subscribe to changes handler
func (s *service) Subscribe(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Subscribe")
	c, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade conn")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer c.Close()

	cnt := 0
	ch := make(chan internalRequest)
	connMu.Lock()
	s.connections[r.RemoteAddr] = ch
	connMu.Unlock()
	defer func() {
		connMu.Lock()
		defer connMu.Unlock()
		delete(s.connections, r.RemoteAddr)
	}()
	for req := range ch {
		func() {
			cnt++
			log.Info().Msgf("Got %d msg for %v", cnt, r.RemoteAddr)

			if err := c.WriteJSON(req.Req); err != nil {
				log.Error().Err(err).Msg("Failed to write json")
				close(req.ResCh)
				return
			}

			var res model.DataMuseResults
			if err := c.ReadJSON(&res); err != nil {
				log.Error().Err(err).Msg("Failed to read json")
				close(req.ResCh)
				return
			}
			req.ResCh <- res
		}()
	}
}

// Publish commands handler
func (s *service) PublishCommand(w http.ResponseWriter, r *http.Request) {
	keys, ok := r.URL.Query()["word"]

	if !ok || len(keys[0]) < 1 {
		log.Error().Msg("Url Param 'word' is missing")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	word := keys[0]

	req := internalRequest{
		Req: model.DataMuseRequest{Word: word},
	}

	var fromMuseApi model.DataMuseResults

	for k, v := range s.connections {
		v := v
		k := k
		resCh := make(chan model.DataMuseResults)
		req.ResCh = resCh
		v <- req
		log.Info().Msg("Sent")
		fromMuseApi, ok = <-req.ResCh
		log.Info().Msgf("Get: %#v", fromMuseApi)
		if !ok {
			log.Error().Msgf("Failed to do req for: %v", k)
			continue
		}
		break
	}

	if fromMuseApi == nil {
		log.Error().Msg("There is no ready processor for your response")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	// Sending to pubsub
	go func() {
		for _, item := range fromMuseApi {
			s.resChan <- item
		}
		log.Info().Msgf("Sent to pubsub channel")
	}()

	// Sending to user
	forUser, err := json.Marshal(fromMuseApi)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal results")
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	log.Info().Msgf("Sended to user, %v", forUser)
	w.WriteHeader(http.StatusOK)
	w.Write(forUser)

}

// Check server handler
func (s *service) Check(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Health check")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	var opts options
	if _, err := flags.Parse(&opts); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize env")
	}

	// Initialize pub sub dataClient
	dataClient, err := pubsub.NewClient(opts.ProjectID, opts.DataTopicName, opts.DataSubName, 60*time.Second)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize pubsub dataClient")
	}

	s := service{
		dataClient:  dataClient,
		resChan:     make(chan model.DataMuseResult),
		upgrader:    websocket.Upgrader{},
		connections: make(map[string]chan internalRequest),
	}
	go func() {
		s.HandleResults()
	}()

	// Initialize server
	r := chi.NewRouter()
	r.Get("/command", s.PublishCommand)
	r.HandleFunc("/subscribe", s.Subscribe)
	r.Get("/health", s.Check)

	srv := http.Server{
		Addr:    fmt.Sprintf(":%v", opts.Port),
		Handler: r,
	}

	log.Info().Msg("Start to serve")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal().Err(err).Msg("Failed to listen and serve")
	}
}

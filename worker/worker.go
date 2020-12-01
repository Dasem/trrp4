package main

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog/log"
	uuid "github.com/satori/go.uuid"

	"github.com/Dasem/trrp4/pubsub"
)

var saveTimeout = 5 * time.Second

type options struct {
	ProjectID string `long:"projectID" env:"PROJECT_ID" required:"true" default:"refined-byte-297215"`
	TopicName string `long:"TopicName" env:"TOPIC_NAME" required:"true" default:"synonyms"`
	SubName   string `long:"SubName" env:"SUB_NAME" required:"true" default:"synonyms-sub"`
}

type server struct {
	firestore *firestore.Client
	pubsub    *pubsub.Client
}

// handleMsg handles messages from pubsub and pass it to firestore
func (s *server) handleMsg(ctx context.Context, data []byte) (bool, error) {
	// Unmarshal data from pubsub to Machine
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal msg")
		return true, err
	}

	// Save Machine to firestore
	if err := s.saveResults(ctx, msg); err != nil {
		log.Error().Err(err).Msg("Failed to save machine")
		return false, err
	}

	return true, nil
}

func (s *server) saveResults(ctx context.Context, data map[string]interface{}) error {

	key := uuid.NewV4().String()

	ctx, cancel := context.WithTimeout(ctx, saveTimeout)
	defer cancel()
	if _, err := s.firestore.Collection("results").Doc(key).Set(ctx, data); err != nil {
		return err
	}

	return nil
}

func main() {
	var opts options
	if _, err := flags.Parse(&opts); err != nil {
		return
	}

	log.Info().Msgf("ProjectID: %v Topic: %v Sub: %v", opts.ProjectID, opts.TopicName, opts.SubName)

	srv := server{}

	// Get a Firestore firestore.
	ctx := context.Background()
	var err error
	srv.firestore, err = firestore.NewClient(ctx, opts.ProjectID)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create firestore")
	}
	// Close firestore when done.
	defer srv.firestore.Close()

	// Initialize pubsub client
	srv.pubsub, err = pubsub.NewClient(opts.ProjectID, opts.TopicName, opts.SubName, saveTimeout)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize pubsub client")
	}

	log.Info().Msg("Start to consume")
	// Start to listen events
	if err := srv.pubsub.Consume(ctx, srv.handleMsg); err != nil {
		log.Fatal().Err(err).Msg("Failed to handle msgs")
	}
}

package model

import "time"

// Example: {"word":"tinnitus","score":51691,"tags":["syn","n"]}
type DataMuseResult struct {
	Word        string    `json:"word"`
	Score       int       `json:"score"`
	Tags        []string  `json:"tags"`
	RequestDate time.Time `json:"request_date"`
}

type ForUserResult struct {
	Word string   `json:"word"`
	Tags []string `json:"tags"`
}

type ForUserResults []ForUserResult

type DataMuseResults []DataMuseResult

type DataMuseRequest struct {
	Word string `json:"word"`
}

type InternalRequest struct {
	ResCh chan DataMuseResult
	Req   DataMuseRequest
}

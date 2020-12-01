package model

type AgentDataRes struct {
	BotId        string           `json:"bot_id"`
	Err          string           `json:"err,omitempty"`
	EndpointData DataMuseResult   `json:"endpoint_data"`
	Def          SourceDefinition `json:"def"`
	RequestTime  string           `json:"request_time"`
	RequestId    string           `json:"request_id"`
	StatusCode   int              `json:"status_code"`
}

// Example: {"word":"tinnitus","score":51691,"tags":["syn","n"]}
type DataMuseResult struct {
	Word  string   `json:"word"`
	Score int      `json:"score"`
	Tags  []string `json:"tags"`
}

type AgentDataReq struct {
	RequestId string           `json:"request_id"`
	Def       SourceDefinition `json:"def"`
}

type SourceDefinition struct {
	Endpoint   string            `json:"endpoint"`
	HttpMethod string            `json:"http_method"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body,omitempty"`
	Timeout    string            `json:"timeout"`
}

type InternalRequest struct {
	ResCh chan AgentDataRes
	Req   AgentDataReq
}

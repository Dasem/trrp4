package models

type AgentDataRes struct {
	BotId        string           `json:"bot_id"`
	Err          string           `json:"err,omitempty"`
	EndpointData string           `json:"endpoint_data"`
	Def          SourceDefinition `json:"def"`
	RequestTime  string           `json:"request_time"`
	RequestId    string           `json:"request_id"`
	StatusCode   int              `json:"status_code"`
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

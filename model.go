package gitlabreceiver

type glJobEvent struct {
	Kind           string  `json:"object_kind"`
	Sha            string  `json:"sha"`
	RetriesCount   int     `json:"retries_count"`
	Id             int     `json:"build_id"`
	Name           string  `json:"build_name"`
	Stage          string  `json:"build_stage"`
	Status         string  `json:"build_status"`
	CreatedAt      string  `json:"build_created_at"`
	StartedAt      string  `json:"build_started_at"`
	FinishedAt     string  `json:"build_finished_at"`
	Duration       float64 `json:"build_duration"`
	FailureReason  string  `json:"build_failure_reason"`
	PipelineId     int     `json:"pipeline_id"`
	JobUrl         string
	PipelineUrl    string
	ParentPipeline ParentPipeline `json:"source_pipeline"`
	Repository     Repository     `json:"repository"`
	Project        Project        `json:"project"`
}

type Repository struct {
	Name string `json:"name"`
	Url  string `json:"homepage"`
}

type Project struct {
	Name string `json:"name"`
	Id   int    `json:"id"`
	Path string `json:"path_with_namespace"`
	Url  string `json:"web_url"`
}

type glPipelineEvent struct {
	Kind           string         `json:"object_kind"`
	Pipeline       Pipeline       `json:"object_attributes"`
	Jobs           []Job          `json:"builds"`
	Project        Project        `json:"project"`
	ParentPipeline ParentPipeline `json:"source_pipeline"`
}

type Pipeline struct {
	Id         int    `json:"id"`
	Status     string `json:"status"`
	Duration   int    `json:"duration"`
	Url        string `json:"url"`
	CreatedAt  string `json:"created_at"`
	FinishedAt string `json:"finished_at"`
	Sha        string `json:"sha"`
	Source     string `json:"source"`
}

type ParentPipeline struct {
	Id      int     `json:"pipeline_id"`
	Project Project `json:"project"`
}

type Job struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Stage       string `json:"stage"`
	CreatedAt   string `json:"created_at"`
	StartedAt   string `json:"started_at"`
	FinishedAt  string `json:"finished_at"`
	Url         string
	ProjectPath string
}

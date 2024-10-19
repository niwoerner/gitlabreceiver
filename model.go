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
	User           User           `json:"user"`
	Commit         Commit         `json:"commit"`
}

type Pipeline struct {
	Id             int         `json:"id"`
	Status         string      `json:"status"`
	Url            string      `json:"url"`
	CreatedAt      string      `json:"created_at"`
	FinishedAt     string      `json:"finished_at"`
	Sha            string      `json:"sha"`
	Source         string      `json:"source"`
	Duration       int         `json:"duration"`
	QueuedDuration int         `json:"queued_duration"`
	Variables      []Variables `json:"variables"`
}

type Variables struct {
	Key   string `json:"key"`
	Value string `json:"value"`
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
	Runner      Runner      `json:"runner"`
	Environment Environment `json:"environment"`
}

type User struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type Commit struct {
	ID        string `json:"id"`
	Message   string `json:"message"`
	Title     string `json:"title"`
	Timestamp string `json:"timestamp"`
	URL       string `json:"url"`
	Author    Author `json:"author"`
}

type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Runner struct {
	Id          int      `json:"id"`
	Description string   `json:"description"`
	Type        string   `json:"runner_type"`
	IsActive    bool     `json:"active"`
	IsShared    bool     `json:"is_shared"`
	Tags        []string `json:"tags"`
}

type Environment struct {
	Name string `json:"name"`
}

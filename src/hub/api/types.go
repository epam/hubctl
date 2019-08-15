package api

import "time"

type ApiErrors struct {
	Errors []ApiError
}

type ApiError struct {
	Type   string
	Source string
	Detail string
	Meta   struct {
		Stack string
		Data  ApiErrors // nested API call
		// validation error
		SchemaPath string `json:"schemaPath"`
		Message    string
		Params     map[string]interface{}
	}
}

type CloudAccount struct {
	Id               string
	Name             string
	Kind             string
	Status           string
	BaseDomain       string `json:"baseDomain"`
	Parameters       []Parameter
	TeamsPermissions []Team `json:"teamsPermissions"`
}

type AwsSecurityCredentials struct {
	Cloud        string
	AccessKey    string
	SecretKey    string
	SessionToken string
	Ttl          int
}

type CloudAccountRequest struct {
	Name        string            `json:"name"`
	Kind        string            `json:"kind"`
	Parameters  []Parameter       `json:"parameters,omitempty"`
	Credentials map[string]string `json:"credentials,omitempty"`
}

type Parameter struct {
	Name      string      `json:"name"`
	Kind      string      `json:"kind,omitempty"`
	Value     interface{} `json:"value,omitempty"`
	From      string      `json:"from,omitempty"`
	Component string      `json:"component,omitempty"`
	Origin    string      `json:"origin,omitempty"`
	Messenger string      `json:"messenger,omitempty"`
}

type Secret struct {
	Name   string
	Kind   string
	Values map[string]string
}

type Output struct {
	Name      string      `json:"name"`
	Component string      `json:"component,omitempty"`
	Kind      string      `json:"kind,omitempty"`
	Value     interface{} `json:"value"`
	Brief     string      `json:"brief,omitempty"`
	Messenger string      `json:"messenger,omitempty"`
}

type Provider struct {
	Kind       string      `json:"kind"`
	Name       string      `json:"name"`
	Provides   []string    `json:"provides,omitempty"`
	Parameters []Parameter `json:"parameters,omitempty"`
}

type Team struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type Environment struct {
	Id               string
	Name             string
	Description      string
	Tags             []string
	CloudAccount     string `json:"cloudAccount"`
	Parameters       []Parameter
	Providers        []Provider
	TeamsPermissions []Team `json:"teamsPermissions"`
}

type EnvironmentRequest struct {
	Name         string      `json:"name"`
	Description  string      `json:"description,omitempty"`
	Tags         []string    `json:"tags,omitempty"`
	CloudAccount string      `json:"cloudAccount"`
	Parameters   []Parameter `json:"parameters"` // TODO omitempty as soon as API is ready
	Providers    []Provider  `json:"providers"`
}

type StackComponent struct {
	Name        string
	Brief       string
	Description string
}

type BaseStack struct {
	Id         string
	Name       string
	Brief      string
	Tags       []string
	Components []StackComponent
	Parameters []Parameter
}

type StackRef struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type GitRef struct {
	Public   string `json:"public"`
	Template *struct {
		Ref string `json:"ref"`
	} `json:"template,omitempty"`
	K8s *struct {
		Ref string `json:"ref"`
	} `json:"k8s,omitempty"`
}

type StackTemplate struct {
	Id                string
	Name              string
	Description       string
	Status            string
	Tags              []string
	Stack             StackRef
	ComponentsEnabled []string `json:"componentsEnabled"`
	Verbs             []string
	GitRemote         GitRef `json:"gitRemote"`
	Parameters        []Parameter
	TeamsPermissions  []Team `json:"teamsPermissions"`
}

type StackTemplateRequest struct {
	Name              string      `json:"name"`
	Description       string      `json:"description,omitempty"`
	Tags              []string    `json:"tags,omitempty"`
	Stack             string      `json:"stack"`
	ComponentsEnabled []string    `json:"componentsEnabled,omitempty"`
	Verbs             []string    `json:"verbs,omitempty"`
	Parameters        []Parameter `json:"parameters,omitempty"`
	TeamsPermissions  []Team      `json:"teamsPermissions,omitempty"`
}

type EnvironmentRef struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain,omitempty"`
}

type StackTemplateRef struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type PlatformRef struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type ComponentStatus struct {
	Name    string            `json:"name"`
	Status  string            `json:"status"`
	Version string            `json:"version,omitempty"`
	Message string            `json:"message,omitempty"`
	Outputs map[string]string `json:"outputs,omitempty"`
}

type LifecyclePhase struct {
	Phase  string `json:"phase"`
	Status string `json:"status"`
}

type InflightOperation struct {
	Id          string                 `json:"id"`
	Operation   string                 `json:"operation"`
	Timestamp   time.Time              `json:"timestamp"`
	Status      string                 `json:"status,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
	Description string                 `json:"description,omitempty"`
	Initiator   string                 `json:"initiator,omitempty"`
	Logs        string                 `json:"logs,omitempty"`
	Phases      []LifecyclePhase       `json:"phases,omitempty"`
}

type TemplateStatus struct {
	Commit  string `json:"commit,omitempty"`
	Ref     string `json:"ref,omitempty"`
	Date    string `json:"date,omitempty"`
	Author  string `json:"author,omitempty"`
	Subject string `json:"subject,omitempty"`
}

type StackInstanceStatus struct {
	Status     string            `json:"status,omitempty"`
	Template   *TemplateStatus   `json:"template,omitempty"`
	K8s        *TemplateStatus   `json:"k8s,omitempty"`
	Components []ComponentStatus `json:"components,omitempty"`
}

type StackInstance struct {
	Id                 string              `json:"id"`
	Name               string              `json:"name"`
	Domain             string              `json:"domain"`
	Description        string              `json:"description,omitempty"`
	Verbs              []string            `json:"verbs,omitempty"`
	Tags               []string            `json:"tags,omitempty"`
	Environment        EnvironmentRef      `json:"environment,omitempty"`
	Stack              StackRef            `json:"stack,omitempty"`
	Template           StackTemplateRef    `json:"template,omitempty"`
	Platform           *PlatformRef        `json:"platform,omitempty"`
	ComponentsEnabled  []string            `json:"componentsEnabled,omitempty"`
	GitRemote          GitRef              `json:"gitRemote,omitempty"`
	Parameters         []Parameter         `json:"parameters,omitempty"`
	Outputs            []Output            `json:"outputs,omitempty"`
	Provides           map[string][]string `json:"provides,omitempty"`
	StateFiles         []string            `json:"stateFiles,omitempty"`
	Status             StackInstanceStatus `json:"status"`
	InflightOperations []InflightOperation `json:"inflightOperations,omitempty"`
}

type StackInstanceRequest struct {
	Name              string      `json:"name"`
	Description       string      `json:"description,omitempty"`
	Tags              []string    `json:"tags,omitempty"`
	Environment       string      `json:"environment"`
	Template          string      `json:"template"`
	Platform          string      `json:"platform,omitempty"`
	ComponentsEnabled []string    `json:"componentsEnabled,omitempty"`
	Parameters        []Parameter `json:"parameters,omitempty"`
}

type StackInstanceDeployResponse struct {
	JobId string `json:"jobId"`
}

type GitRefPatch struct {
	Template *struct {
		Ref string `json:"ref"`
	} `json:"template,omitempty"`
	K8s *struct {
		Ref string `json:"ref"`
	} `json:"k8s,omitempty"`
}

type StackInstancePatch struct {
	Verbs              []string               `json:"verbs,omitempty"`
	ComponentsEnabled  []string               `json:"componentsEnabled,omitempty"`
	StateFiles         []string               `json:"stateFiles,omitempty"`
	Parameters         []Parameter            `json:"parameters,omitempty"`
	GitRemote          *GitRefPatch           `json:"gitRemote,omitempty"`
	Status             *StackInstanceStatus   `json:"status,omitempty"`
	InflightOperations []InflightOperation    `json:"inflightOperations,omitempty"`
	Outputs            []Output               `json:"outputs,omitempty"`
	Provides           map[string][]string    `json:"provides,omitempty"`
	UI                 map[string]interface{} `json:"ui,omitempty"`
}

type Application struct {
	Id               string
	Name             string
	Description      string
	Tags             []string
	Environments     []EnvironmentRef
	Parameters       []Parameter
	GitRemote        GitRef `json:"gitRemote"`
	TeamsPermissions []Team `json:"teamsPermissions"`
}

type License struct {
	Component  string
	LicenseKey string
}

type ServiceAccount struct {
	UserId     string `json:"userId"`
	Name       string `json:"name"`
	GroupId    string `json:"groupId"`
	LoginToken string `json:"loginToken"`
}

type DeploymentKey struct {
	DeploymentKey string `json:"deploymentKey"`
}

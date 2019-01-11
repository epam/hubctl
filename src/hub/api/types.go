package api

type CloudAccount struct {
	Id               string
	Name             string
	Type             string
	Status           string
	BaseDomain       string `json:"baseDomain"`
	Parameters       []Parameter
	TeamsPermissions []Team `json:"teamsPermissions"`
}

type AwsTemporarySecurityCredentials struct {
	Cloud        string
	AccessKey    string
	SecretKey    string
	SessionToken string
	Ttl          int
}

type Parameter struct {
	Name      string      `json:"name"`
	Kind      string      `json:"kind,omitempty"`
	Value     interface{} `json:"value,omitempty"`
	From      string      `json:"from,omitempty"`
	Component string      `json:"component,omitempty"`
	Origin    string      `json:"origin,omitempty"`
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
	TeamsPermissions []Team `json:"teamsPermissions"`
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
}

type StackRef struct {
	Id   string
	Name string
}

type GitRef struct {
	Public   string
	Template struct {
		Ref string
	}
	K8s struct {
		Ref string
	}
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
	Id     string
	Name   string
	Domain string
}

type StackTemplateRef struct {
	Id   string
	Name string
}

type PlatformRef struct {
	Id     string
	Name   string
	Domain string
}

type ComponentStatus struct {
	Name    string            `json:"name"`
	Status  string            `json:"status"`
	Outputs map[string]string `json:"outputs,omitempty"`
}

type LifecyclePhase struct {
	Phase  string `json:"phase"`
	Status string `json:"status"`
}

type InflightOperation struct {
	Operation   string           `json:"operation"`
	Status      string           `json:"status"`
	Description string           `json:"description,omitempty"`
	Logs        string           `json:"logs,omitempty"`
	Phases      []LifecyclePhase `json:"phases,omitempty"`
}

type TemplateStatus struct {
	Commit  string `json:"commit,omitempty"`
	Ref     string `json:"ref,omitempty"`
	Date    string `json:"date,omitempty"`
	Author  string `json:"author,omitempty"`
	Subject string `json:"subject,omitempty"`
}

type StackInstanceStatus struct {
	Status             string              `json:"status,omitempty"`
	Template           *TemplateStatus     `json:"template,omitempty"`
	K8s                *TemplateStatus     `json:"k8s,omitempty"`
	Components         []ComponentStatus   `json:"components,omitempty"`
	InflightOperations []InflightOperation `json:"inflightOperations,omitempty"`
}

type StackInstance struct {
	Id                string
	Name              string
	Domain            string
	Description       string
	Tags              []string
	Environment       EnvironmentRef
	Stack             StackRef
	Template          StackTemplateRef
	Platform          PlatformRef
	ComponentsEnabled []string `json:"componentsEnabled"`
	GitRemote         GitRef   `json:"gitRemote"`
	Parameters        []Parameter
	Outputs           []Output
	Provides          map[string][]string
	StateFiles        []string `json:"stateFiles"`
	Status            StackInstanceStatus
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
	JobId string `json:jobId`
}

type StackInstancePatch struct {
	Status   StackInstanceStatus `json:"status,omitempty"`
	Outputs  []Output            `json:"outputs,omitempty"`
	Provides map[string][]string `json:"provides,omitempty"`
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

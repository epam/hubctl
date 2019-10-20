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
	Id               string      `json:"id"`
	Name             string      `json:"name"`
	Tags             []string    `json:"tags,omitempty"`
	Kind             string      `json:"kind"`
	Status           string      `json:"status"`
	BaseDomain       string      `json:"baseDomain"`
	Parameters       []Parameter `json:"parameters,omitempty"`
	TeamsPermissions []Team      `json:"teamsPermissions"`
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
	Tags        []string          `json:"tags,omitempty"`
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
	Id   string `json:"id,omitempty"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type Environment struct {
	Id               string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	UI               map[string]interface{} `json:"ui,omitempty"`
	CloudAccount     string                 `json:"cloudAccount"`
	Parameters       []Parameter            `json:"parameters,omitempty"`
	Providers        []Provider             `json:"providers,omitempty"`
	TeamsPermissions []Team                 `json:"teamsPermissions"`
}

type EnvironmentRequest struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	UI               map[string]interface{} `json:"ui,omitempty"`
	CloudAccount     string                 `json:"cloudAccount"`
	Parameters       []Parameter            `json:"parameters,omitempty"`
	Providers        []Provider             `json:"providers,omitempty"`
	TeamsPermissions []Team                 `json:"teamsPermissions,omitempty"`
}

type EnvironmentPatch struct {
	Name             string                 `json:"name,omitempty"`
	Description      string                 `json:"description,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	UI               map[string]interface{} `json:"ui,omitempty"`
	Parameters       []Parameter            `json:"parameters,omitempty"`
	Providers        []Provider             `json:"providers,omitempty"`
	TeamsPermissions []Team                 `json:"teamsPermissions,omitempty"`
}

type StackComponent struct {
	Name        string `json:"name"`
	Brief       string `json:"brief,omitempty"`
	Description string `json:"description,omitempty"`
}

type BaseStack struct {
	Id         string           `json:"id"`
	Name       string           `json:"name"`
	Brief      string           `json:"brief,omitempty"`
	Tags       []string         `json:"tags,omitempty"`
	Components []StackComponent `json:"components,omitempty"`
	Parameters []Parameter      `json:"parameters,omitempty"`
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
	Id                string                 `json:"id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description,omitempty"`
	Status            string                 `json:"status"`
	Tags              []string               `json:"tags,omitempty"`
	UI                map[string]interface{} `json:"ui,omitempty"`
	Stack             StackRef               `json:"stack"`
	ComponentsEnabled []string               `json:"componentsEnabled,omitempty"`
	Verbs             []string               `json:"verbs,omitempty"`
	GitRemote         GitRef                 `json:"gitRemote"`
	Parameters        []Parameter            `json:"parameters,omitempty"`
	TeamsPermissions  []Team                 `json:"teamsPermissions"`
}

type StackTemplateRequest struct {
	Name              string                 `json:"name"`
	Description       string                 `json:"description,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	UI                map[string]interface{} `json:"ui,omitempty"`
	Stack             string                 `json:"stack"`
	ComponentsEnabled []string               `json:"componentsEnabled,omitempty"`
	Verbs             []string               `json:"verbs,omitempty"`
	Parameters        []Parameter            `json:"parameters,omitempty"`
	TeamsPermissions  []Team                 `json:"teamsPermissions,omitempty"`
}

type StackTemplatePatch struct {
	Description       string                 `json:"description,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	UI                map[string]interface{} `json:"ui,omitempty"`
	ComponentsEnabled []string               `json:"componentsEnabled,omitempty"`
	Verbs             []string               `json:"verbs,omitempty"`
	Parameters        []Parameter            `json:"parameters,omitempty"`
	TeamsPermissions  []Team                 `json:"teamsPermissions,omitempty"`
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
	// application and overlay
	Provides   map[string][]string `json:"provides,omitempty"`
	StateFiles []string            `json:"stateFiles,omitempty"`
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
	// applications
	JobName        string `json:"jobName,omitempty"`
	PlatformDomain string `json:"platformDomain,omitempty"`
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
	Id                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Domain             string                 `json:"domain"`
	Description        string                 `json:"description,omitempty"`
	Verbs              []string               `json:"verbs,omitempty"`
	Tags               []string               `json:"tags,omitempty"`
	UI                 map[string]interface{} `json:"ui,omitempty"`
	Environment        EnvironmentRef         `json:"environment,omitempty"`
	Stack              StackRef               `json:"stack,omitempty"`
	Template           StackTemplateRef       `json:"template,omitempty"`
	Platform           *PlatformRef           `json:"platform,omitempty"`
	ComponentsEnabled  []string               `json:"componentsEnabled,omitempty"`
	GitRemote          GitRef                 `json:"gitRemote,omitempty"`
	Parameters         []Parameter            `json:"parameters,omitempty"`
	Outputs            []Output               `json:"outputs,omitempty"`
	Provides           map[string][]string    `json:"provides,omitempty"`
	StateFiles         []string               `json:"stateFiles,omitempty"`
	Status             StackInstanceStatus    `json:"status"`
	InflightOperations []InflightOperation    `json:"inflightOperations,omitempty"`
}

type StackInstanceRequest struct {
	Name              string                 `json:"name"`
	Description       string                 `json:"description,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	UI                map[string]interface{} `json:"ui,omitempty"`
	Environment       string                 `json:"environment"`
	Template          string                 `json:"template"`
	Platform          string                 `json:"platform,omitempty"`
	ComponentsEnabled []string               `json:"componentsEnabled,omitempty"`
	Parameters        []Parameter            `json:"parameters,omitempty"`
}

type StackInstanceLifecycleRequest struct {
	Components []string `json:"components,omitempty"`
}

type StackInstanceLifecycleResponse struct {
	Id    string `json:"id"`
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
	Description        string                 `json:"description,omitempty"`
	Tags               []string               `json:"tags,omitempty"`
	UI                 map[string]interface{} `json:"ui,omitempty"`
	Verbs              []string               `json:"verbs,omitempty"`
	ComponentsEnabled  []string               `json:"componentsEnabled,omitempty"`
	StateFiles         []string               `json:"stateFiles,omitempty"`
	Parameters         []Parameter            `json:"parameters,omitempty"`
	GitRemote          *GitRefPatch           `json:"gitRemote,omitempty"`
	Status             *StackInstanceStatus   `json:"status,omitempty"`
	InflightOperations []InflightOperation    `json:"inflightOperations,omitempty"`
	Outputs            []Output               `json:"outputs,omitempty"`
	Provides           map[string][]string    `json:"provides,omitempty"`
}

type StackInstanceRef struct {
	Id         string      `json:"id"`
	Name       string      `json:"name"`
	Domain     string      `json:"domain"`
	Stack      StackRef    `json:"stack,omitempty"`
	Platform   PlatformRef `json:"platform,omitempty"`
	Parameters []Parameter `json:"parameters,omitempty"`
}

type ComponentBackupOutput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ComponentBackup struct {
	Kind      string                  `json:"kind"`
	Status    string                  `json:"status"`
	Timestamp time.Time               `json:"timestamp"`
	Outputs   []ComponentBackupOutput `json:"outputs,omitempty"`
}

type BackupBundle struct {
	Kind       string                     `json:"kind"`
	Status     string                     `json:"status"`
	Timestamp  time.Time                  `json:"timestamp"`
	Components map[string]ComponentBackup `json:"components,omitempty"`
}

type Backup struct {
	Id            string                 `json:"id"`
	Name          string                 `json:"name"`
	Status        string                 `json:"status"`
	Timestamp     time.Time              `json:"timestamp"`
	Components    []string               `json:"components,omitempty"`
	Description   string                 `json:"description,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	UI            map[string]interface{} `json:"ui,omitempty"`
	Environment   EnvironmentRef         `json:"environment"`
	StackInstance StackInstanceRef       `json:"stackInstance"`
	Logs          string                 `json:"logs,omitempty"`
	Bundle        BackupBundle           `json:"bundle"`
}

type BackupRequest struct {
	Name       string   `json:"name"`
	Components []string `json:"components,omitempty"`
}

type Application struct {
	Id                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description,omitempty"`
	Tags               []string               `json:"tags,omitempty"`
	UI                 map[string]interface{} `json:"ui,omitempty"`
	Application        string                 `json:"application"`
	Status             string                 `json:"status"`
	Environment        EnvironmentRef         `json:"environment"`
	Environments       []EnvironmentRef       `json:"environments,omitempty"` // not implemented
	Platform           PlatformRef            `json:"platform"`
	Requires           []string               `json:"requires"`
	Parameters         []Parameter            `json:"parameters,omitempty"`
	Outputs            []Output               `json:"outputs,omitempty"`
	StateFiles         []string               `json:"stateFiles,omitempty"`
	InflightOperations []InflightOperation    `json:"inflightOperations,omitempty"`
}

// unused
type ApplicationRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	UI          map[string]interface{} `json:"ui,omitempty"`
	Application string                 `json:"application"`
	Requires    []string               `json:"requires"`
	Platform    string                 `json:"platform"`
	Parameters  []Parameter            `json:"parameters,omitempty"`
}

type ApplicationPatch struct {
	UI         map[string]interface{} `json:"ui,omitempty"`
	Parameters []Parameter            `json:"parameters,omitempty"`
}

type License struct {
	Component  string `json:"component"`
	LicenseKey string `json:"licenseKey"`
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

type ComponentGitRef struct {
	Remote string `json:"remote"`
	SubDir string `json:"subDir,omitempty"`
}

type Component struct {
	Id               string                 `json:"id,omitempty"`
	QName            string                 `json:"qname"`
	Brief            string                 `json:"brief,omitempty"`
	Description      string                 `json:"description,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	UI               map[string]interface{} `json:"ui,omitempty"`
	Category         string                 `json:"category,omitempty"`
	License          string                 `json:"license,omitempty"`
	Icon             string                 `json:"icon,omitempty"`
	Template         *StackTemplateRef      `json:"template,omitempty"`
	Git              *ComponentGitRef       `json:"git,omitempty"`
	Version          string                 `json:"version,omitempty"`
	Maturity         string                 `json:"maturity,omitempty"`
	Requires         []string               `json:"requires,omitempty"`
	Provides         []string               `json:"provides,omitempty"`
	TeamsPermissions []Team                 `json:"teamsPermissions,omitempty"`
}

type ComponentRequest struct {
	Name             string                 `json:"name"`
	Brief            string                 `json:"brief,omitempty"`
	Description      string                 `json:"description,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	UI               map[string]interface{} `json:"ui,omitempty"`
	Category         string                 `json:"category,omitempty"`
	License          string                 `json:"license,omitempty"`
	Icon             string                 `json:"icon,omitempty"`
	Template         string                 `json:"template,omitempty"`
	SubDir           string                 `json:"subDir,omitempty"`
	Version          string                 `json:"version,omitempty"`
	Maturity         string                 `json:"maturity,omitempty"`
	Requires         []string               `json:"requires,omitempty"`
	Provides         []string               `json:"provides,omitempty"`
	TeamsPermissions []Team                 `json:"teamsPermissions,omitempty"`
}

type ComponentPatch struct {
	Brief            string                 `json:"brief,omitempty"`
	Description      string                 `json:"description,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	UI               map[string]interface{} `json:"ui,omitempty"`
	Category         string                 `json:"category,omitempty"`
	License          string                 `json:"license,omitempty"`
	Icon             string                 `json:"icon,omitempty"`
	Version          string                 `json:"version,omitempty"`
	Maturity         string                 `json:"maturity,omitempty"`
	Requires         []string               `json:"requires,omitempty"`
	Provides         []string               `json:"provides,omitempty"`
	TeamsPermissions []Team                 `json:"teamsPermissions,omitempty"`
}

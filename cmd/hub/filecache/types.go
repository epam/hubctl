package filecache

type AccessTokenBox struct {
	ApiBaseUrl     string
	LoginTokenHash uint64
	AccessToken    string
	RefreshToken   string
}

type Metrics struct {
	Disabled bool
	Host     *string `yaml:",omitempty"`
}

type FileCache struct {
	Version      int
	AccessTokens []AccessTokenBox
	Metrics      Metrics
}

package kube

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/agilestacks/hub/cmd/hub/config"
	"github.com/agilestacks/hub/cmd/hub/parameters"
	"github.com/agilestacks/hub/cmd/hub/state"
	"github.com/agilestacks/hub/cmd/hub/util"
)

func Kubeconfig(filenames []string, providers []string, context string, keepPems bool) {
	state := state.MustParseStateFiles(filenames)

	providerState, provider := findState(state, providers)
	if providerState == nil && config.Verbose {
		log.Printf("There is no state for %v found in state file(s) %v; trying stack parameters", providers, filenames)
	}

	outputs := make(parameters.CapturedOutputs)
	if providerState != nil {
		for _, o := range providerState.CapturedOutputs {
			outputs[o.QName()] = o
		}
	}
	params := make(parameters.LockedParameters)
	for _, p := range state.StackParameters {
		params[p.QName()] = p
	}
	SetupKubernetes(params, provider, outputs, context, config.Force, keepPems)
}

func findState(state *state.StateManifest, providers []string) (*state.StateStep, string) {
	for _, provider := range providers {
		providerState, exist := state.Components[provider]
		if exist {
			return providerState, provider
		}
	}
	return nil, ""
}

type KubeAuthPlugin struct {
	ApiVersion string   `yaml:"apiVersion,omitempty"`
	Command    string   `yaml:"command,omitempty"`
	Args       []string `yaml:"args,omitempty"`
}
type KubeUser struct {
	ClientCertificate     string         `yaml:"client-certificate,omitempty"`
	ClientCertificateData string         `yaml:"client-certificate-data,omitempty"`
	ClientKey             string         `yaml:"client-key,omitempty"`
	ClientKeyData         string         `yaml:"client-key-data,omitempty"`
	Exec                  KubeAuthPlugin `yaml:"exec,omitempty"`
}
type KubeUsers struct {
	Name string   `yaml:"name,omitempty"`
	User KubeUser `yaml:"user,omitempty"`
}
type KubeConfig struct {
	Kind           string            `yaml:"kind,omitempty"`
	ApiVersion     string            `yaml:"apiVersion,omitempty"`
	Preferences    map[string]string `yaml:"preferences,omitempty"`
	CurrentContext string            `yaml:"current-context,omitempty"`
	Clusters       []struct {
		Name    string            `yaml:"name,omitempty"`
		Cluster map[string]string `yaml:"cluster,omitempty"`
	} `yaml:"clusters,omitempty"`
	Contexts []struct {
		Name    string            `yaml:"name,omitempty"`
		Context map[string]string `yaml:"context,omitempty"`
	} `yaml:"contexts,omitempty"`
	Users []KubeUsers `yaml:"users,omitempty"`
}

func kubeconfigFilename() (string, error) {
	filename := os.Getenv("KUBECONFIG")
	if filename != "" {
		return filename, nil
	}

	home := os.Getenv("HOME")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		return "", errors.New("HOME / USERPROFILE, nor KUBECONFIG is set")
	}

	dir := filepath.Join(home, ".kube")
	const defaultModeDir = 0775
	os.MkdirAll(dir, defaultModeDir)

	filename = filepath.Join(dir, "config")
	_, err := os.Stat(filename)
	if err != nil {
		if util.NoSuchFile(err) {
			if config.Verbose {
				log.Printf("Kubeconfig `%s` not found", filename)
			}
		} else {
			return "", fmt.Errorf("Unable to stat `%s` Kubeconfig: %v", filename, err)
		}
	}
	return filename, nil
}

func readKubeconfig(filename string) (*KubeConfig, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("Unable to open `%s`: %v", filename, err)
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("Unable to read `%s`: %v", filename, err)
	}
	var config KubeConfig
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return nil, fmt.Errorf("Unable to unmarshall `%s`: %v", filename, err)
	}
	return &config, nil
}

func writeKubeconfig(filename string, config *KubeConfig) error {
	file, err := os.OpenFile(filename, os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("Unable to open `%s`: %v", filename, err)
	}
	defer file.Close()
	content, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("Unable to marshall `%s`: %v", filename, err)
	}
	_, err = file.Seek(0, os.SEEK_SET)
	if err != nil {
		return fmt.Errorf("Unable to seek `%s`: %v", filename, err)
	}
	wrote, err := file.Write(content)
	if err != nil || wrote != len(content) {
		return fmt.Errorf("Unable to write `%s` (wrote %d bytes): %s", filename, wrote, util.Errors2(err))
	}
	err = file.Truncate(int64(wrote))
	if err != nil {
		return fmt.Errorf("Unable to truncate `%s`: %v", filename, err)
	}
	return nil
}

func addHeptioUser(configFilename, user, cluster string) {
	config, err := readKubeconfig(configFilename)
	if err != nil {
		log.Fatalf("%v", err)
	}

	auth := KubeUser{Exec: KubeAuthPlugin{
		ApiVersion: "client.authentication.k8s.io/v1alpha1",
		Command:    "aws-iam-authenticator",
		Args:       []string{"token", "-i", cluster},
	}}

	found := false
	for i := range config.Users {
		u := &config.Users[i]
		if u.Name == user {
			u.User = auth
			found = true
			break
		}
	}
	if !found {
		config.Users = append(config.Users, KubeUsers{Name: user, User: auth})
	}

	err = writeKubeconfig(configFilename, config)
	if err != nil {
		log.Fatalf("%v", err)
	}
}

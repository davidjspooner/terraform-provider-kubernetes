package kresource

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type KubeCtlNamedCluster struct {
	Name    string `yaml:"name"`
	Cluster struct {
		CertificateAuthorityData string `yaml:"certificate-authority-data"`
		Server                   string `yaml:"server"`
	} `yaml:"cluster"`
}

type KubeCtlNamedContext struct {
	Name    string `yaml:"name"`
	Context struct {
		Cluster string `yaml:"cluster"`
		User    string `yaml:"user"`
	} `yaml:"context"`
}

type KubeCtlNamedUser struct {
	Name string `yaml:"name"`
	User struct {
		ClientCertificateData string `yaml:"client-certificate-data"`
		ClientKeyData         string `yaml:"client-key-data"`
	} `yaml:"user"`
}

type KubeCtlConfig struct {
	APIVersion     string                `yaml:"apiVersion"`
	Kind           string                `yaml:"kind"`
	Clusters       []KubeCtlNamedCluster `yaml:"clusters"`
	Contexts       []KubeCtlNamedContext `yaml:"contexts"`
	CurrentContext string                `yaml:"current-context"`
	Preferences    struct{}              `yaml:"preferences"`
	Users          []KubeCtlNamedUser    `yaml:"users"`
}

// Load reads a YAML file into the KubernetesConfig struct.
func (config *KubeCtlConfig) Load(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening KubernetesConfig: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("error reading KubernetesConfig: %w", err)
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return fmt.Errorf("error unmarshaling YAML: %w", err)
	}
	err = config.Validate()
	if err != nil {
		return fmt.Errorf("error validating KubernetesConfig: %w", err)
	}

	return nil
}

// Validate checks if the KubernetesConfig struct is valid.
func (config *KubeCtlConfig) Validate() error {
	if config.APIVersion != "v1" {
		return fmt.Errorf("invalid API version: %s", config.APIVersion)
	}

	if config.Kind != "Config" {
		return fmt.Errorf("invalid kind: %s", config.Kind)
	}

	return nil
}

func (config *KubeCtlConfig) FindCluster(name string) int {
	for i, cluster := range config.Clusters {
		if cluster.Name == name {
			return i
		}
	}
	return -1
}
func (config *KubeCtlConfig) FindContext(name string) int {
	for i, context := range config.Contexts {
		if context.Name == name {
			return i
		}
	}
	return -1
}
func (config *KubeCtlConfig) FindUser(name string) int {
	for i, user := range config.Users {
		if user.Name == name {
			return i
		}
	}
	return -1
}

func (config *KubeCtlConfig) WriteToFile(filename string) error {

	filename, err := ExpandEnv(filename)
	if err != nil {
		return fmt.Errorf("error expanding template filename: %w", err)
	}

	newData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshaling YAML: %w", err)
	}

	f, err := os.Open(filename)
	var backupName string
	if err == nil {
		oldData, _ := io.ReadAll(f)
		if string(oldData) == string(newData) {
			return nil
		}
		f.Close() //explicit release the file lock
		now := time.Now().Format("20060102150405")
		backupName = filename + "." + now + ".bak"
		os.Rename(filename, backupName)
	}

	f, err = os.Create(filename)
	if err != nil {
		f.Close() //explicit release the file lock
		if backupName != "" {
			os.Rename(backupName, filename)
		}
		return fmt.Errorf("error creating file: %w", err)
	}
	_, err = f.Write(newData)
	f.Close() //explicit release the file lock

	if err != nil {
		if backupName != "" {
			os.Rename(backupName, filename)
		}
		return fmt.Errorf("error writing file: %w", err)
	}

	err = os.Chmod(filename, 0600)
	if err != nil {
		return fmt.Errorf("error changing file permissions: %w", err)
	}

	return nil
}

// -----------------------------------------------------------------------------

type K8sConfigPair struct {
	Template KubeCtlConfig
	Target   KubeCtlConfig
}

func (pair *K8sConfigPair) LoadConfigs(templateFilename string, targetFilename string) error {

	templateFilename, err := ExpandEnv(templateFilename)
	if err != nil {
		return fmt.Errorf("error expanding template filename: %w", err)
	}

	err = pair.Template.Load(templateFilename)
	if err != nil {
		curDir, _ := os.Getwd()
		return fmt.Errorf("error loading template: %w (curr dir =%q)", err, curDir)
		//pair.Template = Config{
		//	APIVersion: "v1",
		//	Kind:       "Config",
		//}
	}

	targetFilename, err = ExpandEnv(targetFilename)
	if err != nil {
		return fmt.Errorf("error expanding target filename: %w", err)
	}
	err = pair.Target.Load(targetFilename)
	if err != nil {
		pair.Target = KubeCtlConfig{
			APIVersion: "v1",
			Kind:       "Config",
		}
	}

	return nil
}

func (pair *K8sConfigPair) UpdateTemplate(name, server string) error {
	if len(pair.Template.Clusters) == 0 {
		return fmt.Errorf("no clusters in template")
	}
	if len(pair.Template.Clusters) > 1 {
		return fmt.Errorf("multiple clusters in template")
	}

	serverUrl, err := url.Parse(server)
	if err != nil {
		return fmt.Errorf("error parsing server URL: %w", err)
	}
	serverUrl.Scheme = "https"
	if serverUrl.Port() == "" {
		serverUrl.Host = serverUrl.Host + ":16443" //or :443?
	}

	pair.Template.Clusters[0].Cluster.Server = serverUrl.String()
	pair.Template.Clusters[0].Name = name
	for i := range pair.Template.Contexts {
		pair.Template.Contexts[i].Context.Cluster = name
	}
	return nil
}

func (pair *K8sConfigPair) RemoveClusterFromTarget(name string) error {

	i := pair.Target.FindCluster(name)
	if i >= 0 {
		pair.Target.Clusters = append(pair.Target.Clusters[:i], pair.Target.Clusters[i+1:]...)
	}

	for i := 0; i < len(pair.Target.Contexts); {
		context := &pair.Target.Contexts[i]
		if context.Context.Cluster == name {
			pair.Target.Contexts = append(pair.Target.Contexts[:i], pair.Target.Contexts[i+1:]...)
		} else {
			i++
		}
	}
	for i := 0; i < len(pair.Target.Users); {
		user := &pair.Target.Users[i]
		if strings.HasSuffix(user.Name, "@"+name) {
			pair.Target.Users = append(pair.Target.Users[:i], pair.Target.Users[i+1:]...)
		} else {
			i++
		}
	}
	if len(pair.Target.Clusters) == 0 {
		pair.Target.CurrentContext = ""
	} else {
		pair.Target.CurrentContext = pair.Target.Contexts[0].Name
	}
	return nil
}

func (pair *K8sConfigPair) MergeTemplateIntoTarget() error {
	if len(pair.Template.Clusters) == 0 {
		return fmt.Errorf("no clusters in template")
	}
	if len(pair.Template.Clusters) > 1 {
		return fmt.Errorf("multiple clusters in template")
	}

	clusterName := pair.Template.Clusters[0].Name
	i := pair.Target.FindCluster(clusterName)
	if i >= 0 {
		pair.Target.Clusters[i].Cluster.CertificateAuthorityData = pair.Template.Clusters[0].Cluster.CertificateAuthorityData
		pair.Target.Clusters[i].Cluster.Server = pair.Template.Clusters[0].Cluster.Server
	} else {
		pair.Target.Clusters = append(pair.Target.Clusters, pair.Template.Clusters[0])
	}

	for _, context := range pair.Template.Contexts {
		context.Context.Cluster = clusterName
		context.Context.User += "@" + clusterName
		context.Name += "-" + clusterName
		if i := pair.Target.FindContext(context.Name); i >= 0 {
			pair.Target.Contexts[i].Context = context.Context
		} else {
			pair.Target.Contexts = append(pair.Target.Contexts, context)
		}
	}

	for _, user := range pair.Template.Users {
		user.Name += "@" + clusterName
		if i := pair.Target.FindUser(user.Name); i >= 0 {
			pair.Target.Users[i] = user
		} else {
			pair.Target.Users = append(pair.Target.Users, user)
		}
	}

	pair.Target.CurrentContext = pair.Template.CurrentContext + "-" + clusterName

	return nil
}

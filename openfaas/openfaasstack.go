package openfaasstack

import (
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const (
	stackFile   = "stack.yaml"
	serviceFile = "openfaas.yml"
)

type Limit struct {
	Memory string `yaml:"memory"`
}

type Function struct {
	Name        string
	Handler     string `yaml:"handler"`
	Description string
	Runtime     string `yaml:"lang"`
	MemorySize  string
	limits      Limit `yaml:"limits"`
}

type Functions map[string]Function

type Service struct {
	Functions Functions
}

type StackInfo struct {
	Name    string `yaml:"name"`
	Project string `yaml:"project"`
	Stage   string `yaml:"stage"`
}

type OpenFaaSStack struct {
	stackInfo  StackInfo
	path       string
	gatewayUrl string
	Functions  []*Function
}

func New(path string, gatewayUrl string) (*OpenFaaSStack, error) {
	stackInfoFile, err := ioutil.ReadFile(filepath.Join(path, stackFile))
	if err != nil {
		return nil, err
	}

	var info StackInfo
	err = yaml.Unmarshal(stackInfoFile, &info)
	if err != nil {
		return nil, err
	}

	stack := OpenFaaSStack{stackInfo: info, path: path, gatewayUrl: gatewayUrl}

	serviceFileRaw, err := ioutil.ReadFile(filepath.Join(path, serviceFile))
	if err != nil {
		return nil, err
	}

	var service Service
	err = yaml.Unmarshal(serviceFileRaw, &service)
	if err != nil {
		return nil, err
	}

	for k, v := range service.Functions {
		v.Name = k
		v.MemorySize = v.limits.Memory
		stack.Functions = append(stack.Functions, &v)
	}

	return &stack, nil
}

// TODO (Sam)
func (s *OpenFaaSStack) DeployStack() error {
	return nil
}

// TODO (Sam)
func (s *OpenFaaSStack) RemoveStack() error {
	return nil
}

func (s *OpenFaaSStack) StackId() string {
	return s.stackInfo.Name
}

func (s *OpenFaaSStack) Project() string {
	return s.stackInfo.Project
}

func (s *OpenFaaSStack) Stage() string {
	return s.stackInfo.Stage
}

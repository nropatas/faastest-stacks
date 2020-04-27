package openfaasstack

import (
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/nropatas/faastest-stacks/utils"
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
}

type FunctionSpec struct {
	Function `yaml:",inline"`
	Limits   Limit `yaml:"limits"`
}

type Functions map[string]FunctionSpec

type Service struct {
	Functions
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
		v.MemorySize = v.Limits.Memory
		stack.Functions = append(stack.Functions, &v.Function)
	}

	return &stack, nil
}

func (s *OpenFaaSStack) DeployStack() error {
	_, _, err := utils.ExecCmd([]string{}, s.path,
		"faas-cli", "deploy", "-g", s.gatewayUrl, "-f", serviceFile)
	if err != nil {
		return err
	}

	time.Sleep(5 * time.Second)

	return nil
}

func (s *OpenFaaSStack) RemoveStack() error {
	_, _, err := utils.ExecCmd([]string{}, s.path,
		"faas-cli", "remove", "-g", s.gatewayUrl, "-f", serviceFile)
	if err != nil {
		return err
	}

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

package knativestack

import (
	"io/ioutil"
	"path/filepath"

	"github.com/nropatas/faastest-stacks/utils"
	"gopkg.in/yaml.v2"
)

const (
	stackFile    = "stack.yaml"
	kserviceFile = "service.yml"
)

type KserviceMeta struct {
	Name string `yaml:"name"`
}

type Kservice struct {
	Meta KserviceMeta `yaml:"metadata"`
}

type Function struct {
	Name        string
	Handler     string
	Description string
	Runtime     string
	MemorySize  string
	dirName     string
}

type StackInfo struct {
	Name    string `yaml:"name"`
	Project string `yaml:"project"`
	Stage   string `yaml:"stage"`
}

type KnativeStack struct {
	stackInfo StackInfo
	path      string
	Functions []*Function
}

func New(path string) (*KnativeStack, error) {
	stackInfoFile, err := ioutil.ReadFile(filepath.Join(path, stackFile))
	if err != nil {
		return nil, err
	}

	var info StackInfo
	err = yaml.Unmarshal(stackInfoFile, &info)
	if err != nil {
		return nil, err
	}

	stack := KnativeStack{stackInfo: info, path: path}

	functions, _ := ioutil.ReadDir(path)
	for _, function := range functions {
		if function.IsDir() {
			file, err := ioutil.ReadFile(filepath.Join(path, function.Name(), kserviceFile))
			if err != nil {
				return nil, err
			}

			var service Kservice
			err = yaml.Unmarshal(file, &service)
			if err != nil {
				return nil, err
			}

			stack.Functions = append(stack.Functions, &Function{Name: service.Meta.Name, dirName: function.Name()})
		}
	}

	return &stack, nil
}

func (s *KnativeStack) DeployStack() error {
	for _, function := range s.Functions {
		// Deploy the function
		_, _, err := utils.ExecCmd([]string{"KUBECONFIG=\"/root/.kube/kubeconfig_knative\""}, filepath.Join(s.path, function.dirName),
			"kubectl", "apply", "-f", kserviceFile)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *KnativeStack) RemoveStack() error {
	for _, function := range s.Functions {
		_, _, err := utils.ExecCmd([]string{"KUBECONFIG=\"/root/.kube/kubeconfig_knative\""}, filepath.Join(s.path, function.dirName),
			"kubectl", "delete", "-f", kserviceFile)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *KnativeStack) StackId() string {
	return s.stackInfo.Name
}

func (s *KnativeStack) Project() string {
	return s.stackInfo.Project
}

func (s *KnativeStack) Stage() string {
	return s.stackInfo.Stage
}

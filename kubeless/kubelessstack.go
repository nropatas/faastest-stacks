package kubelessstack

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/nropatas/faastest-stacks/utils"
	"gopkg.in/yaml.v2"
)

type Function struct {
	Name        string `yaml:"name"`
	Handler     string `yaml:"handler"`
	Description string `yaml:"description"`
	Runtime     string `yaml:"runtime"`
	MemorySize  string `yaml:"memory"`
}

const (
	stackFile = "stack.yaml"
	specFile  = "kubeless.yml"
)

type FunctionSpec struct {
	Function      `yaml:",inline"`
	Path          string `yaml:"path"`
	CPU           string `yaml:"cpu"`
	MinScale      string `yaml:"min-scale"`
	MaxScale      string `yaml:"max-scale"`
	TargetCPU     string `yaml:"target-cpu"`
	Env           string `yaml:"env"`
	File          string `yaml:"file"`
	Dependencties string `yaml:"dependencies"`
}

type Functions map[string]FunctionSpec

type Spec struct {
	Hostname     string `yaml:"hostname"`
	File         string `yaml:"file"`
	Dependencies string `yaml:"dependencies"`
	Functions
}

type StackInfo struct {
	Name    string `yaml:"name"`
	Project string `yaml:"project"`
	Stage   string `yaml:"stage"`
}

type KubelessStack struct {
	stackInfo StackInfo
	path      string
	spec      Spec
	Functions []*Function
}

func New(path string) (*KubelessStack, error) {
	stackInfoFile, err := ioutil.ReadFile(filepath.Join(path, stackFile))
	if err != nil {
		return nil, err
	}

	var info StackInfo
	err = yaml.Unmarshal(stackInfoFile, &info)
	if err != nil {
		return nil, err
	}

	specFileRaw, err := ioutil.ReadFile(filepath.Join(path, specFile))
	if err != nil {
		return nil, err
	}

	var spec Spec
	err = yaml.Unmarshal(specFileRaw, &spec)
	if err != nil {
		return nil, err
	}

	stack := KubelessStack{stackInfo: info, path: path, spec: spec}

	for _, v := range spec.Functions {
		stack.Functions = append(stack.Functions, &v.Function)
	}

	return &stack, nil
}

func (s *KubelessStack) DeployStack() error {
	_, _, err := utils.ExecCmd([]string{}, s.path, "npm", "i")
	if err != nil {
		return err
	}

	for _, f := range s.spec.Functions {
		handlerFile := s.spec.File
		if f.File != "" {
			handlerFile = f.File
		}

		deployArgs := []string{"function", "deploy", f.Name, "-r", f.Runtime, "-f", handlerFile,
			"--handler", f.Handler, "--cpu", f.CPU, "--memory", f.MemorySize}

		dependencies := s.spec.Dependencies
		if f.Dependencties != "" {
			dependencies = f.Dependencties
		}

		if dependencies != "" {
			deployArgs = append(deployArgs, "--dependencies", dependencies)
		}

		if f.Env != "" {
			deployArgs = append(deployArgs, "--env", f.Env)
		}

		_, _, err = utils.ExecCmd([]string{}, s.path, "kubeless", deployArgs...)
		if err != nil {
			return err
		}

		// Check if the function is ready
		stdout := ""
		for strings.Compare(stdout, "True") != 0 {
			time.Sleep(5 * time.Second)

			stdout, _, err = utils.ExecCmd([]string{}, s.path,
				"/bin/sh", "-c", fmt.Sprintf("kubectl get pods -l function=%s -o jsonpath='{.items[0].status.conditions[1].status}' --kubeconfig /app/kubeconfigs/kubeconfig_kubeless", f.Name))
			if err != nil {
				return err
			}
		}

		_, _, err = utils.ExecCmd([]string{}, s.path,
			"kubeless", "trigger", "http", "create", f.Name, "--function-name", f.Name, "--hostname", s.spec.Hostname,
			"--path", f.Path)
		if err != nil {
			return err
		}

		_, _, err = utils.ExecCmd([]string{}, s.path,
			"kubeless", "autoscale", "create", f.Name, "--metric", "cpu",
			"--min", f.MinScale, "--max", f.MaxScale, "--value", f.TargetCPU)
		if err != nil {
			return err
		}
	}

	time.Sleep(5 * time.Second)

	return nil
}

func (s *KubelessStack) RemoveStack() error {
	for _, f := range s.spec.Functions {
		utils.ExecCmd([]string{}, s.path, "kubeless", "function", "delete", f.Name)
	}

	return nil
}

func (s *KubelessStack) StackId() string {
	return s.stackInfo.Name
}

func (s *KubelessStack) Project() string {
	return s.stackInfo.Project
}

func (s *KubelessStack) Stage() string {
	return s.stackInfo.Stage
}

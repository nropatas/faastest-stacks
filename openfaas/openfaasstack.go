package openfaasstack

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
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

type Autoscaling struct {
	Min       string `yaml:"min"`
	Max       string `yaml:"max"`
	TargetCPU string `yaml:"target-cpu"`
}

type Function struct {
	Name        string
	Handler     string `yaml:"handler"`
	Description string
	Runtime     string `yaml:"lang"`
	MemorySize  string
}

type FunctionSpec struct {
	Function     `yaml:",inline"`
	Limits       Limit `yaml:"limits"`
	*Autoscaling `yaml:"autoscaling"`
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
	service    Service
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

	serviceFileRaw, err := ioutil.ReadFile(filepath.Join(path, serviceFile))
	if err != nil {
		return nil, err
	}

	var service Service
	err = yaml.Unmarshal(serviceFileRaw, &service)
	if err != nil {
		return nil, err
	}

	stack := OpenFaaSStack{stackInfo: info, path: path, gatewayUrl: gatewayUrl, service: service}

	for k, v := range service.Functions {
		v.Name = k
		v.MemorySize = v.Limits.Memory
		stack.Functions = append(stack.Functions, &Function{
			Name:        v.Name,
			Description: v.Description,
			Handler:     v.Handler,
			Runtime:     v.Runtime,
			MemorySize:  v.MemorySize})
	}

	return &stack, nil
}

func (s *OpenFaaSStack) DeployStack() error {
	_, _, err := utils.ExecCmd([]string{}, s.path,
		"faas-cli", "deploy", "-g", s.gatewayUrl, "-f", serviceFile)
	if err != nil {
		return err
	}

	for _, f := range s.service.Functions {
		if f.Autoscaling != nil {
			_, _, err = utils.ExecCmd([]string{}, s.path,
				"/bin/sh", "-c", fmt.Sprintf("kubectl autoscale deployment -n openfaas-fn %s --cpu-percent %s --min %s --max %s --kubeconfig /app/kubeconfigs/kubeconfig_openfaas", f.Name, f.TargetCPU, f.Min, f.Max))
			if err != nil {
				return err
			}
		}
	}

	// Check if all functions are ready
	funcStatuses := make([]bool, len(s.Functions))
	for !utils.IsAllTrue(funcStatuses) {
		time.Sleep(5 * time.Second)

		for i, f := range s.Functions {
			stdout, _, err := utils.ExecCmd([]string{}, s.path,
				"faas-cli", "describe", "-g", s.gatewayUrl, f.Name)
			if err != nil {
				return err
			}

			isReady := !strings.Contains(stdout, "Not Ready")
			funcStatuses[i] = isReady

			// Don't check more functions. Continue waiting right away.
			if !isReady {
				break
			}
		}
	}

	return nil
}

func (s *OpenFaaSStack) RemoveStack() error {
	_, _, err := utils.ExecCmd([]string{}, s.path,
		"faas-cli", "remove", "-g", s.gatewayUrl, "-f", serviceFile)
	if err != nil {
		return err
	}

	for _, f := range s.service.Functions {
		if f.Autoscaling != nil {
			_, _, err = utils.ExecCmd([]string{}, s.path,
				"/bin/sh", "-c", fmt.Sprintf("kubectl delete hpa -n openfaas-fn %s --kubeconfig /app/kubeconfigs/kubeconfig_openfaas", f.Name))
			if err != nil {
				return err
			}
		}
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

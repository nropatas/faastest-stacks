package knativestack

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

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

	kserviceFileRaw, err := ioutil.ReadFile(filepath.Join(path, kserviceFile))
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(kserviceFileRaw)
	decoder := yaml.NewDecoder(reader)

	var ksvc Kservice
	for decoder.Decode(&ksvc) == nil {
		stack.Functions = append(stack.Functions, &Function{Name: ksvc.Meta.Name})
	}

	return &stack, nil
}

func (s *KnativeStack) DeployStack() error {
	_, _, err := utils.ExecCmd([]string{}, s.path,
		"/bin/sh", "-c", fmt.Sprintf("kubectl apply -f %s --kubeconfig /app/kubeconfigs/kubeconfig_knative", kserviceFile))
	if err != nil {
		return err
	}

	// Check if all functions are ready
	funcStatuses := make([]bool, len(s.Functions))
	for !utils.IsAllTrue(funcStatuses) {
		time.Sleep(5 * time.Second)

		for i, f := range s.Functions {
			stdout, _, err := utils.ExecCmd([]string{}, s.path,
				"/bin/sh", "-c", fmt.Sprintf("kubectl get ksvc %s -o jsonpath='{.status.conditions[1].status}' --kubeconfig /app/kubeconfigs/kubeconfig_knative", f.Name))
			if err != nil {
				return err
			}

			isReady := strings.Compare(stdout, "True") == 0
			funcStatuses[i] = isReady

			// Don't check more functions. Continue waiting right away.
			if !isReady {
				break
			}
		}
	}

	return nil
}

func (s *KnativeStack) RemoveStack() error {
	_, _, err := utils.ExecCmd([]string{}, s.path,
		"/bin/sh", "-c", fmt.Sprintf("kubectl delete -f %s --kubeconfig /app/kubeconfigs/kubeconfig_knative", kserviceFile))
	if err != nil {
		return err
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

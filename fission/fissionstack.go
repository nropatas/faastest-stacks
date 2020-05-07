package fissionstack

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/nropatas/faastest-stacks/utils"
	"gopkg.in/yaml.v2"
)

const (
	stackFile = "stack.yaml"
	specFile  = "fission.yml"
)

type Function struct {
	Name        string `yaml:"name"`
	Handler     string `yaml:"handler"`
	Description string `yaml:"description"`
	Runtime     string
	MemorySize  string `yaml:"maxmemory"`
}

type FunctionSpec struct {
	Function  `yaml:",inline"`
	Minmemory string `yaml:"minmemory"`
	Mincpu    string `yaml:"mincpu"`
	Maxcpu    string `yaml:"maxcpu"`
	Minscale  string `yaml:"minscale"`
	Maxscale  string `yaml:"maxscale"`
	Targetcpu string `yaml:"targetcpu"`
}

type Functions map[string]FunctionSpec

type EnvSpec struct {
	Name  string `yaml:"name"`
	Image string `yaml:"image"`
}

type Env struct {
	Env EnvSpec `yaml:"env"`
	Functions
}

type Envs map[string]Env

type Spec struct {
	Envs
}

type StackInfo struct {
	Name    string `yaml:"name"`
	Project string `yaml:"project"`
	Stage   string `yaml:"stage"`
}

type FissionStack struct {
	stackInfo StackInfo
	path      string
	spec      Spec
	Functions []*Function
}

func New(path string) (*FissionStack, error) {
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

	stack := FissionStack{stackInfo: info, path: path, spec: spec}

	for _, env := range spec.Envs {
		for _, f := range env.Functions {
			stack.Functions = append(stack.Functions, &f.Function)
		}
	}

	return &stack, nil
}

func (s *FissionStack) DeployStack() error {
	_, _, err := utils.ExecCmd([]string{}, s.path, "npm", "i")
	if err != nil {
		return err
	}

	for _, env := range s.spec.Envs {
		_, _, err = utils.ExecCmd([]string{}, s.path,
			"fission", "env", "create", "--name", env.Env.Name, "--image", env.Env.Image)
		if err != nil {
			return err
		}

		for _, f := range env.Functions {
			_, _, err = utils.ExecCmd([]string{}, s.path,
				"fission", "fn", "create", "--name", f.Name, "--env", env.Env.Name, "--code", f.Handler, "--executortype", "newdeploy",
				"--mincpu", f.Mincpu, "--maxcpu", f.Maxcpu, "--minmemory", f.Minmemory, "--maxmemory", f.MemorySize,
				"--minscale", f.Minscale, "--maxscale", f.Maxscale, "--targetcpu", f.Targetcpu)
			if err != nil {
				return err
			}

			_, _, err = utils.ExecCmd([]string{}, s.path,
				"fission", "route", "create", "--method", "POST", "--url", fmt.Sprintf("/%s", f.Name), "--function", f.Name, "--name", f.Name)
			if err != nil {
				return err
			}
		}
	}

	time.Sleep(10 * time.Second)

	return nil
}

func (s *FissionStack) RemoveStack() error {
	for _, f := range s.Functions {
		utils.ExecCmd([]string{}, s.path, "fission", "httptrigger", "delete", "--name", f.Name)
		utils.ExecCmd([]string{}, s.path, "fission", "fn", "delete", "--name", f.Name)
	}

	utils.ExecCmd([]string{}, s.path, "fission", "pkg", "delete", "--orphan")

	for _, env := range s.spec.Envs {
		utils.ExecCmd([]string{}, s.path, "fission", "env", "delete", "--name", env.Env.Name)
	}

	return nil
}

func (s *FissionStack) StackId() string {
	return s.stackInfo.Name
}

func (s *FissionStack) Project() string {
	return s.stackInfo.Project
}

func (s *FissionStack) Stage() string {
	return s.stackInfo.Stage
}

package fissionstack

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

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
	minmemory   string `yaml:"minmemory"`
	mincpu      string `yaml:"mincpu"`
	maxcpu      string `yaml:"maxcpu"`
	minscale    string `yaml:"minscale"`
	maxscale    string `yaml:"maxscale"`
	targetcpu   string `yaml:"targetcpu"`
}

type Functions map[string]Function

type Environment struct {
	name  string `yaml:"name"`
	image string `yaml:"image"`
}

type Spec struct {
	env       Environment `yaml:"env"`
	functions Functions
}

type StackInfo struct {
	Name    string `yaml:"name"`
	Project string `yaml:"project"`
	Stage   string `yaml:"stage"`
}

type FissionStack struct {
	stackInfo   StackInfo
	path        string
	environment Environment
	Functions   []*Function
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

	stack := FissionStack{stackInfo: info, path: path, environment: spec.env}

	for _, v := range spec.functions {
		stack.Functions = append(stack.Functions, &v)
	}

	return &stack, nil
}

func (s *FissionStack) DeployStack() error {
	_, _, err := utils.ExecCmd([]string{}, s.path,
		"fission", "env", "create", "--name", s.environment.name, "--image", s.environment.image)
	if err != nil {
		return err
	}

	for _, f := range s.Functions {
		_, _, err = utils.ExecCmd([]string{}, s.path,
			"fission", "fn", "create", "--name", f.Name, "--env", s.environment.name, "--code", f.Handler, "--executortype", "newdeploy",
			"--mincpu", f.mincpu, "--maxcpu", f.maxcpu, "--minmemory", f.minmemory, "--maxmemory", f.MemorySize,
			"--minscale", f.minscale, "--maxscale", f.maxscale, "--targetcpu", f.targetcpu)
		if err != nil {
			return err
		}

		_, _, err = utils.ExecCmd([]string{}, s.path,
			"fission", "route", "create", "--method", "POST", "--url", fmt.Sprintf("/%s", f.Name), "--function", f.Name, "--name", f.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *FissionStack) RemoveStack() error {
	for _, f := range s.Functions {
		utils.ExecCmd([]string{}, s.path, "fission", "httptrigger", "delete", "--name", f.Name)
		utils.ExecCmd([]string{}, s.path, "fission", "fn", "delete", "--name", f.Name)
	}

	utils.ExecCmd([]string{}, s.path, "fission", "pkg", "delete", "--orphan")
	utils.ExecCmd([]string{}, s.path, "fission", "env", "delete", "--name", s.environment.name)

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

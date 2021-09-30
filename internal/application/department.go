package application

import (
	"fmt"
	"regexp"
)

// DepartmentConfig is the config-file format for departments.
type DepartmentConfig struct {
	Departments []Department `json:"departments"`
}

// Department is a department.
type Department struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	County     string `json:"county"`
	Block      string `json:"block"`
	blockRegex *regexp.Regexp
}

// DepartmentService handles department information.
type DepartmentService struct {
	Departments []Department
}

// Load the department information.
func (s *DepartmentService) Load(config DepartmentConfig) error {
	s.Departments = config.Departments
	for i := range s.Departments {
		if s.Departments[i].Block != "" {
			var err error
			s.Departments[i].blockRegex, err = regexp.Compile(s.Departments[i].Block)
			if err != nil {
				return fmt.Errorf("could not compile %q: %v", s.Departments[i].Block, err)
			}
		}
	}
	return nil
}

// GetDepartmentByID returns the department with the given ID.
// If no such department exists, then this returns nil.
func (s *DepartmentService) GetDepartmentByID(id string) *Department {
	for _, department := range s.Departments {
		if department.ID == id {
			d := department
			return &d
		}
	}
	return nil
}

// GetDepartmentByBlock returns the department for the given block.
// If no such department exists, then this returns nil.
func (s *DepartmentService) GetDepartmentByBlock(block string) *Department {
	for _, department := range s.Departments {
		if department.blockRegex != nil {
			if department.blockRegex.MatchString(block) {
				d := department
				return &d
			}
		}
	}
	return nil
}

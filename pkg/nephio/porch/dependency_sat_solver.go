/*

Copyright 2025.



Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.

You may obtain a copy of the License at



    http://www.apache.org/licenses/LICENSE-2.0



Unless required by applicable law or agreed to in writing, software

distributed under the License is distributed on an "AS IS" BASIS,

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions and

limitations under the License.

*/

package porch

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	semver "github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SATSolverImpl implements a SAT solver for complex constraint satisfaction in dependency resolution.

type SATSolverImpl struct {
	logger logr.Logger

	variables map[string]*SATVariable

	clauses []*SATClause

	assignments map[string]bool

	unitPropagations []string

	conflictAnalysis *ConflictAnalysis

	backtrackLevel int

	statistics *SATStatistics

	config *SATSolverConfigImpl

	mu sync.RWMutex
}

// SATVariable represents a variable in the SAT problem.

type SATVariable struct {
	ID string

	PackageRef *PackageReference

	Version string

	Domain []string // Possible versions

	Constraints []*VersionConstraint

	DecisionLevel int

	Reason *SATClause

	Watched []*SATClause
}

// SATClause represents a clause in the SAT problem.

type SATClause struct {
	ID string

	Literals []*SATLiteral

	Satisfied bool

	Watched [2]int // Two-watched literals optimization

	Learned bool

	Activity float64
}

// SATLiteral represents a literal in a SAT clause.

type SATLiteral struct {
	Variable *SATVariable

	Positive bool

	Version string
}

// SATStatistics tracks solver performance.

type SATStatistics struct {
	Variables int

	Clauses int

	Decisions int

	Conflicts int

	Propagations int

	Backtracks int

	LearnedClauses int

	Restarts int

	SolveTime time.Duration

	MemoryUsage int64
}

// SATSolverConfig configures the SAT solver.

// SATSolverConfigImpl is the internal implementation config
// (SATSolverConfig is defined in dependency_types.go)
type SATSolverConfigImpl struct {
	MaxDecisions int

	MaxConflicts int

	RestartThreshold int

	DecayFactor float64

	ClauseDecay float64

	RandomSeed int64

	Timeout time.Duration

	EnableLearning bool

	EnableVSIDS bool // Variable State Independent Decaying Sum
}

// NewSATSolver creates a new SAT solver instance.

func NewSATSolver(config *SATSolverConfigImpl) SATSolver {
	if config == nil {
		config = &SATSolverConfigImpl{
			MaxDecisions: 10000,

			MaxConflicts: 1000,

			RestartThreshold: 100,

			DecayFactor: 0.95,

			ClauseDecay: 0.999,

			Timeout: 5 * time.Minute,

			EnableLearning: true,

			EnableVSIDS: true,
		}
	}

	return &SATSolverImpl{
		logger: log.Log.WithName("sat-solver"),

		variables: make(map[string]*SATVariable),

		clauses: []*SATClause{},

		assignments: make(map[string]bool),

		statistics: &SATStatistics{},

		config: config,
	}
}

// Solve attempts to find a satisfying assignment for the given requirements.

func (s *SATSolverImpl) Solve(ctx context.Context, requirements []*VersionRequirement) (*VersionSolution, error) {
	s.logger.Info("Starting SAT solver", "requirements", len(requirements))

	startTime := time.Now()

	// Initialize the problem.

	if err := s.initialize(requirements); err != nil {
		return nil, fmt.Errorf("failed to initialize SAT problem: %w", err)
	}

	// Create timeout context.

	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)

	defer cancel()

	// Main DPLL loop with CDCL (Conflict-Driven Clause Learning).

	solution, err := s.dpllWithCDCL(ctx)
	if err != nil {
		return nil, err
	}

	s.statistics.SolveTime = time.Since(startTime)

	// Convert SAT solution to version solution.

	versionSolution := s.convertToVersionSolution(solution)

	versionSolution.Statistics = &SolutionStatistics{
		Variables: s.statistics.Variables,

		Clauses: s.statistics.Clauses,

		Decisions: s.statistics.Decisions,

		Conflicts: s.statistics.Conflicts,

		Propagations: s.statistics.Propagations,

		Backtracks: s.statistics.Backtracks,

		LearnedClauses: s.statistics.LearnedClauses,

		SolveTime: s.statistics.SolveTime,
	}

	s.logger.Info("SAT solver completed",

		"success", versionSolution.Success,

		"decisions", s.statistics.Decisions,

		"conflicts", s.statistics.Conflicts,

		"duration", s.statistics.SolveTime)

	return versionSolution, nil
}

// initialize sets up the SAT problem from version requirements.

func (s *SATSolverImpl) initialize(requirements []*VersionRequirement) error {
	s.mu.Lock()

	defer s.mu.Unlock()

	// Create variables for each package version.

	for _, req := range requirements {
		if err := s.createVariables(req); err != nil {
			return err
		}
	}

	// Create clauses from constraints.

	for _, req := range requirements {
		if err := s.createClauses(req); err != nil {
			return err
		}
	}

	// Add conflict clauses for incompatible versions.

	if err := s.addConflictClauses(); err != nil {
		return err
	}

	// Initialize watched literals.

	s.initializeWatchedLiterals()

	s.statistics.Variables = len(s.variables)

	s.statistics.Clauses = len(s.clauses)

	return nil
}

// createVariables creates SAT variables for package versions.

func (s *SATSolverImpl) createVariables(req *VersionRequirement) error {
	packageKey := req.PackageRef.GetPackageKey()

	// Get available versions for the package.

	availableVersions, err := s.getAvailableVersions(req.PackageRef)
	if err != nil {
		return fmt.Errorf("failed to get available versions for %s: %w", packageKey, err)
	}

	// Filter versions based on constraints.

	validVersions := s.filterVersionsByConstraints(availableVersions, req.Constraints)

	if len(validVersions) == 0 {
		return fmt.Errorf("no valid versions found for %s with constraints", packageKey)
	}

	// Create a variable for each valid version.

	for _, version := range validVersions {

		varID := fmt.Sprintf("%s@%s", packageKey, version)

		s.variables[varID] = &SATVariable{
			ID: varID,

			PackageRef: req.PackageRef,

			Version: version,

			Domain: validVersions,

			Constraints: req.Constraints,

			Watched: []*SATClause{},
		}

	}

	return nil
}

// createClauses creates SAT clauses from version constraints.

func (s *SATSolverImpl) createClauses(req *VersionRequirement) error {
	packageKey := req.PackageRef.GetPackageKey()

	// At least one version must be selected (positive clause).

	var literals []*SATLiteral

	for varID, variable := range s.variables {
		if strings.HasPrefix(varID, packageKey+"@") {
			literals = append(literals, &SATLiteral{
				Variable: variable,

				Positive: true,

				Version: variable.Version,
			})
		}
	}

	if len(literals) > 0 {

		clause := &SATClause{
			ID: fmt.Sprintf("at-least-one-%s", packageKey),

			Literals: literals,

			Learned: false,

			Activity: 1.0,
		}

		s.clauses = append(s.clauses, clause)

	}

	// At most one version can be selected (negative clauses).

	for i := range len(literals) {
		for j := i + 1; j < len(literals); j++ {

			clause := &SATClause{
				ID: fmt.Sprintf("at-most-one-%s-%d-%d", packageKey, i, j),

				Literals: []*SATLiteral{
					{Variable: literals[i].Variable, Positive: false, Version: literals[i].Version},

					{Variable: literals[j].Variable, Positive: false, Version: literals[j].Version},
				},

				Learned: false,

				Activity: 1.0,
			}

			s.clauses = append(s.clauses, clause)

		}
	}

	return nil
}

// dpllWithCDCL implements DPLL with Conflict-Driven Clause Learning.

func (s *SATSolverImpl) dpllWithCDCL(ctx context.Context) (map[string]bool, error) {
	decisionLevel := 0

	restartCount := 0

	for {

		// Check context cancellation.

		select {

		case <-ctx.Done():

			return nil, fmt.Errorf("SAT solver timeout: %w", ctx.Err())

		default:

		}

		// Unit propagation.

		conflict := s.unitPropagate(decisionLevel)

		if conflict != nil {

			s.statistics.Conflicts++

			if decisionLevel == 0 {
				// Conflict at root level - UNSAT.

				return nil, fmt.Errorf("unsatisfiable constraints")
			}

			// Analyze conflict and learn clause.

			if s.config.EnableLearning {

				learnedClause, backtrackLevel := s.analyzeConflict(conflict, decisionLevel)

				if learnedClause != nil {

					s.clauses = append(s.clauses, learnedClause)

					s.statistics.LearnedClauses++

				}

				decisionLevel = s.backtrack(backtrackLevel)

				s.statistics.Backtracks++

			} else {

				// Simple backtrack.

				decisionLevel = s.backtrack(decisionLevel - 1)

				s.statistics.Backtracks++

			}

			// Check for restart.

			if s.statistics.Conflicts > 0 && s.statistics.Conflicts%s.config.RestartThreshold == 0 {

				restartCount++

				s.statistics.Restarts++

				decisionLevel = s.restart()

			}

			continue

		}

		// Check if all variables are assigned.

		if s.allVariablesAssigned() {
			// Found satisfying assignment.

			return s.assignments, nil
		}

		// Make a decision.

		variable := s.selectVariable()

		if variable == nil {
			return nil, fmt.Errorf("no unassigned variables found")
		}

		decisionLevel++

		s.statistics.Decisions++

		s.assign(variable, true, decisionLevel, nil)

		// Check decision limits.

		if s.statistics.Decisions > s.config.MaxDecisions {
			return nil, fmt.Errorf("exceeded maximum decisions limit")
		}

		if s.statistics.Conflicts > s.config.MaxConflicts {
			return nil, fmt.Errorf("exceeded maximum conflicts limit")
		}

	}
}

// unitPropagate performs unit propagation.

func (s *SATSolverImpl) unitPropagate(decisionLevel int) *SATClause {
	changed := true

	for changed {

		changed = false

		for _, clause := range s.clauses {

			if clause.Satisfied {
				continue
			}

			unassigned := []*SATLiteral{}

			hasTrue := false

			for _, literal := range clause.Literals {

				value, assigned := s.getAssignment(literal.Variable)

				if !assigned {
					unassigned = append(unassigned, literal)
				} else if (literal.Positive && value) || (!literal.Positive && !value) {

					hasTrue = true

					clause.Satisfied = true

					break

				}

			}

			if hasTrue {
				continue
			}

			if len(unassigned) == 0 {
				// Conflict detected.

				return clause
			}

			if len(unassigned) == 1 {

				// Unit clause - must assign this literal.

				literal := unassigned[0]

				s.assign(literal.Variable, literal.Positive, decisionLevel, clause)

				s.statistics.Propagations++

				changed = true

			}

		}

	}

	return nil
}

// selectVariable selects the next variable to assign using VSIDS heuristic.

func (s *SATSolverImpl) selectVariable() *SATVariable {
	if !s.config.EnableVSIDS {

		// Simple selection: first unassigned variable.

		for _, variable := range s.variables {
			if _, assigned := s.assignments[variable.ID]; !assigned {
				return variable
			}
		}

		return nil

	}

	// VSIDS: Variable State Independent Decaying Sum.

	var bestVariable *SATVariable

	bestActivity := 0.0

	for _, variable := range s.variables {
		if _, assigned := s.assignments[variable.ID]; !assigned {

			activity := s.calculateVariableActivity(variable)

			if activity > bestActivity {

				bestActivity = activity

				bestVariable = variable

			}

		}
	}

	return bestVariable
}

// analyzeConflict performs conflict analysis and learns a new clause.

func (s *SATSolverImpl) analyzeConflict(conflict *SATClause, decisionLevel int) (*SATClause, int) {
	// Implement 1-UIP (First Unique Implication Point) learning.

	seen := make(map[string]bool)

	learned := []*SATLiteral{}

	backtrackLevel := 0

	// Start with the conflict clause.

	for _, literal := range conflict.Literals {
		if literal.Variable.DecisionLevel == decisionLevel {
			seen[literal.Variable.ID] = true
		} else if literal.Variable.DecisionLevel > 0 {

			learned = append(learned, &SATLiteral{
				Variable: literal.Variable,

				Positive: !literal.Positive,

				Version: literal.Version,
			})

			if literal.Variable.DecisionLevel > backtrackLevel {
				backtrackLevel = literal.Variable.DecisionLevel
			}

		}
	}

	// Build learned clause.

	if len(learned) > 0 {

		learnedClause := &SATClause{
			ID: fmt.Sprintf("learned-%d", s.statistics.LearnedClauses),

			Literals: learned,

			Learned: true,

			Activity: 1.0,
		}

		return learnedClause, backtrackLevel

	}

	return nil, decisionLevel - 1
}

// Helper methods.

func (s *SATSolverImpl) getAvailableVersions(ref *PackageReference) ([]string, error) {
	// In a real implementation, this would query the package repository.

	// For now, return mock versions.

	return []string{"1.0.0", "1.1.0", "1.2.0", "2.0.0", "2.1.0"}, nil
}

func (s *SATSolverImpl) filterVersionsByConstraints(versions []string, constraints []*VersionConstraint) []string {
	valid := []string{}

	for _, version := range versions {
		if s.satisfiesConstraints(version, constraints) {
			valid = append(valid, version)
		}
	}

	return valid
}

func (s *SATSolverImpl) satisfiesConstraints(version string, constraints []*VersionConstraint) bool {
	v, err := semver.NewVersion(version)
	if err != nil {
		return false
	}

	for _, constraint := range constraints {

		c, err := semver.NewConstraint(s.constraintToSemver(constraint))
		if err != nil {
			return false
		}

		if !c.Check(v) {
			return false
		}

	}

	return true
}

func (s *SATSolverImpl) constraintToSemver(constraint *VersionConstraint) string {
	// Convert our constraint format to semver constraint string.

	switch constraint.Operator {

	case ConstraintOperatorEquals:

		return "= " + constraint.Version

	case ConstraintOperatorGreaterThan:

		return "> " + constraint.Version

	case ConstraintOperatorGreaterEquals:

		return ">= " + constraint.Version

	case ConstraintOperatorLessThan:

		return "< " + constraint.Version

	case ConstraintOperatorLessEquals:

		return "<= " + constraint.Version

	case ConstraintOperatorTilde:

		return "~ " + constraint.Version

	case ConstraintOperatorCaret:

		return "^ " + constraint.Version

	default:

		return constraint.Version

	}
}

func (s *SATSolverImpl) addConflictClauses() error {
	// Add clauses for known incompatible versions.

	// This would be based on compatibility matrix in production.

	return nil
}

func (s *SATSolverImpl) initializeWatchedLiterals() {
	// Initialize two-watched literals for each clause.

	for _, clause := range s.clauses {
		if len(clause.Literals) >= 2 {

			clause.Watched[0] = 0

			clause.Watched[1] = 1

		}
	}
}

func (s *SATSolverImpl) assign(variable *SATVariable, value bool, level int, reason *SATClause) {
	s.assignments[variable.ID] = value

	variable.DecisionLevel = level

	variable.Reason = reason
}

func (s *SATSolverImpl) getAssignment(variable *SATVariable) (bool, bool) {
	value, assigned := s.assignments[variable.ID]

	return value, assigned
}

func (s *SATSolverImpl) allVariablesAssigned() bool {
	return len(s.assignments) == len(s.variables)
}

func (s *SATSolverImpl) backtrack(level int) int {
	// Remove assignments made after the given level.

	for _, variable := range s.variables {
		if variable.DecisionLevel > level {

			delete(s.assignments, variable.ID)

			variable.DecisionLevel = -1

			variable.Reason = nil

		}
	}

	// Reset clause satisfaction status.

	for _, clause := range s.clauses {
		clause.Satisfied = false
	}

	return level
}

func (s *SATSolverImpl) restart() int {
	// Complete restart - unassign all variables.

	s.assignments = make(map[string]bool)

	for _, variable := range s.variables {

		variable.DecisionLevel = -1

		variable.Reason = nil

	}

	for _, clause := range s.clauses {
		clause.Satisfied = false
	}

	return 0
}

func (s *SATSolverImpl) calculateVariableActivity(variable *SATVariable) float64 {
	activity := 0.0

	for _, clause := range variable.Watched {
		activity += clause.Activity
	}

	return activity
}

func (s *SATSolverImpl) convertToVersionSolution(assignments map[string]bool) *VersionSolution {
	solution := &VersionSolution{
		Success: assignments != nil,

		Solutions: make(map[string]*VersionSelection),

		Algorithm: "DPLL-CDCL",
	}

	if assignments == nil {
		return solution
	}

	// Group assignments by package.

	packageVersions := make(map[string][]string)

	for varID, assigned := range assignments {
		if assigned {

			parts := strings.Split(varID, "@")

			if len(parts) == 2 {

				packageKey := parts[0]

				version := parts[1]

				packageVersions[packageKey] = append(packageVersions[packageKey], version)

			}

		}
	}

	// Create version selections.

	for packageKey, versions := range packageVersions {
		if len(versions) > 0 {

			// Sort versions and select the latest.

			sort.Strings(versions)

			selectedVersion := versions[len(versions)-1]

			solution.Solutions[packageKey] = &VersionSelection{
				SelectedVersion: selectedVersion,

				Reason: SelectionReasonConstraintSatisfaction,

				Alternatives: versions,

				Confidence: 1.0,
			}

		}
	}

	return solution
}

// Close cleans up the SAT solver.

func (s *SATSolverImpl) Close() {
	s.logger.Info("Closing SAT solver")

	// Cleanup resources if needed.
}

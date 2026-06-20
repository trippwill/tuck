package plan

import (
	"errors"

	"github.com/trippwill/tuck/internal/apperr"
	"github.com/trippwill/tuck/internal/domain"
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/state"
)

type OperationOptions struct {
	Context  string
	SourceID string
	Apply    bool
}

type Operation struct {
	source   state.Source
	scope    domain.TargetScope
	registry state.Registry
	apply    bool
	plan     Plan
}

func NewOperation(options OperationOptions) (Operation, error) {
	selection, err := domain.SelectActive(domain.SelectionOptions{
		SourceID:    options.SourceID,
		Context:     options.Context,
		RequireHome: true,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNoHome) {
			return Operation{}, apperr.AppErrMsg(ErrApply, err.Error())
		}
		return Operation{}, err
	}
	return Operation{
		source:   selection.Source,
		scope:    selection.Scope,
		registry: selection.Registry,
		apply:    options.Apply,
		plan:     NewPlan(selection.Scope.Context, options.Apply),
	}, nil
}

func NewPlan(context string, apply bool) Plan {
	return Plan{
		Context:   context,
		DryRun:    !apply,
		Applied:   false,
		Privilege: Privilege{Required: false},
		Actions:   []Action{},
		Conflicts: []Conflict{},
	}
}

func (op Operation) Source() state.Source {
	return op.source
}

func (op Operation) Scope() domain.TargetScope {
	return op.scope
}

func (op Operation) Registry() state.Registry {
	return op.registry
}

func (op *Operation) ResolvePackages(refs []string, all bool) ([]packages.Resolved, error) {
	resolvedPackages, err := packages.Resolve(op.source, op.scope.Context, refs, all)
	if err != nil {
		return nil, err
	}
	op.addPackages(resolvedPackages)
	return resolvedPackages, nil
}

func (op *Operation) AddPackage(identity string) {
	op.plan.Packages = append(op.plan.Packages, identity)
}

func (op *Operation) SetPackages(identities ...string) {
	op.plan.Packages = append([]string(nil), identities...)
}

func (op *Operation) AddAction(action Action) {
	op.plan.Actions = append(op.plan.Actions, action)
}

func (op *Operation) AddActions(actions ...Action) {
	op.plan.Actions = append(op.plan.Actions, actions...)
}

func (op *Operation) AddConflict(conflict Conflict) {
	op.plan.Conflicts = append(op.plan.Conflicts, conflict)
}

func (op Operation) HasConflicts() bool {
	return len(op.plan.Conflicts) > 0
}

func (op *Operation) addPackages(resolvedPackages []packages.Resolved) {
	for _, pkg := range resolvedPackages {
		op.plan.Packages = append(op.plan.Packages, pkg.Identity.String())
	}
}

func (op *Operation) Finalize() (Plan, error) {
	if len(op.plan.Conflicts) > 0 {
		op.plan.Actions = []Action{}
		return op.plan, nil
	}
	markPrivilege(&op.plan)
	if op.apply {
		if privilegeDenied(op.plan) {
			return op.plan, nil
		}
		if err := Apply(op.plan); err != nil {
			return op.plan, err
		}
		op.plan.Applied = true
		op.plan.DryRun = false
	}
	return op.plan, nil
}

func markPrivilege(plan *Plan) {
	if plan.Context != domain.ContextRoot || len(plan.Actions) == 0 {
		return
	}
	satisfied := hasRootPrivilege()
	plan.Privilege = Privilege{
		Required:  true,
		Satisfied: &satisfied,
		Reason:    "root-context write",
	}
}

func privilegeDenied(planned Plan) bool {
	return !planned.DryRun && planned.Privilege.Required && planned.Privilege.Satisfied != nil && !*planned.Privilege.Satisfied
}

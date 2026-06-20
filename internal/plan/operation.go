package plan

import (
	"errors"

	"github.com/trippwill/tuck/internal/domain"
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/state"
)

type operationOptions struct {
	Context    string
	TargetRoot string
	SourceID   string
	Apply      bool
}

type operation struct {
	source   state.Source
	scope    domain.TargetScope
	registry state.Registry
	apply    bool
	plan     Plan
}

func newOperation(options operationOptions) (operation, error) {
	selection, err := domain.SelectActive(domain.SelectionOptions{
		SourceID:    options.SourceID,
		Context:     options.Context,
		TargetRoot:  options.TargetRoot,
		RequireHome: true,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNoHome) {
			return operation{}, AppErrMsg(ErrApply, err.Error())
		}
		return operation{}, err
	}
	return operation{
		source:   selection.Source,
		scope:    selection.Scope,
		registry: selection.Registry,
		apply:    options.Apply,
		plan:     newPlan(selection.Scope.Context, options.Apply),
	}, nil
}

func newPlan(context string, apply bool) Plan {
	return Plan{
		Context:   context,
		DryRun:    !apply,
		Applied:   false,
		Privilege: Privilege{Required: false},
		Actions:   []Action{},
		Conflicts: []Conflict{},
	}
}

func (op *operation) resolvePackages(refs []string, all bool) ([]packages.Resolved, error) {
	resolvedPackages, err := packages.Resolve(op.source, op.scope.Context, refs, all)
	if err != nil {
		return nil, err
	}
	op.addPackages(resolvedPackages)
	return resolvedPackages, nil
}

func (op *operation) addPackages(resolvedPackages []packages.Resolved) {
	for _, pkg := range resolvedPackages {
		op.plan.Packages = append(op.plan.Packages, pkg.Identity.String())
	}
}

func (op *operation) finalize() (Plan, error) {
	if len(op.plan.Conflicts) > 0 {
		op.plan.Actions = []Action{}
		return op.plan, nil
	}
	markPrivilege(&op.plan)
	if op.apply {
		if privilegeDenied(op.plan) {
			return op.plan, AppErrMsg(ErrPrivilegeRequired, "root-context write requires elevated privileges")
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

func privilegeDenied(plan Plan) bool {
	return plan.Privilege.Required && plan.Privilege.Satisfied != nil && !*plan.Privilege.Satisfied
}

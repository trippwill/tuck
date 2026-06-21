# Contributing

Thanks for helping improve tuck.

## Development

Use the repo's mise tasks:

```sh
mise run check
```

Useful focused commands:

```sh
mise run test
mise run test:unit
mise run test:accept
mise run coverage
```

## Expectations

- Keep changes small and user-observable.
- Prefer the standard library and existing tooling over new dependencies.
- Add acceptance coverage for command behavior and unit tests for pure logic.
- Mutating command behavior should stay plan-by-default and require `--apply`.
- Use Conventional Commits, for example `feat(cli): add source add`.

## AI-assisted contributions

AI assistance is welcome, but contributors are responsible for the result. Review
the diff, run the relevant checks, and disclose material AI assistance when it is
useful context for reviewers.


# Contributing Guidelines

Welcome to Hubctl. We are excited you have joined our community. This document will help you get started as a contributor.

## Issues Tracking

All activities has been tracked as a Github issues in several repositories and have been attached to [this](https://github.com/orgs/epam/projects/8/views/2) Github project.

All issues attached to the project are in the status *"To Do"* and are ready to be picked up by contributors.

Once issue has been picked up it goes into *"In Progress"* status.
If issue has been blocked it should be labeled as `Help wanted`. This allows to track issues that are blocked and need help from other contributors.

Once feature have been fully implemented it goes into *"Done"* status. and __not__ closed. Issue closed only when it has been released.

Status *"Done"* also means the feature branch have been integrated into `develop` branch (see blow about branching strategy).

Issue from *"Done"* status goes into maybe moved *"In Progress"* status (or even reopened) if additional work have been identified. We do not want to keep duplicated issues (or regressions in case of bugs) as separate issues. Instead, please re-energize the existing issue.

If duplicated issue have been identified it should be closed with labeld `duplicated` and reference to the original issue.

## Branching Strategy

There are three types of branches:

- `main` (or `master`) branch - is the stable branch. It is used for releases only. No direct commits are allowed into this branch.
- `develop` branch - is the integration branch. It contains the latest greatest features. Force push is not allowed here. Allows only lineal history. Changes merged on behalf of PRs (unless trivial, such as typo in readme file). Only merge-rebase workflow is allowed.
- `feature branch` - created on behalf of feature/bugfix.

    - Should be branched from `develop` branch.
    - Should be merged back into `develop` branch.
    - Should deleted once merged.
    - Should contain issue number in the name. Before merged, a PR should be created. PR should be reviewed by at least one other developer.
    - Force push is allowed here

> Feature branches should be continuously integrated into `develop` branch. It should not take more than few days to integrate.

### Reporting Bugs

If you find a bug, please report it by opening an issue. Please include as much information as possible, including:

```markdown
# Issue Title

## Steps to reproduce

Also contains information about environment (OS, shell, Golang variables etc.)

## Actual behavior

## Expected behavior
```

Such issue should be labeled as a `bug`.

### Suggesting Enhancements

If you have an idea for a new feature or an enhancement, please report it by opening an issue (labeled by `enhancement`). Please include as much information as possible, including:

```markdown
# Issue Title

Current situation

Use case description and why it is important

Proposed solution

Alternatives (if any applicable)

Definition of Done in the form of checlikst

- [ ] Lorem
- [ ] Ipsum
- [ ] Dolor
```

Steps like "update documentation on [hubct.io](http://github.com/epam/hubctl.io) etc are good candidates for the checklist items.

If enhancement is rather large, then separate items in the checklist can be broken into multiple issues. Goal is not too have a one issue that will be in "In Progress" status for several weeks. Instead, we want to have a number of issues that can be implemented in a two days max.

### Pushing Changes

1. Pick and issue and update its status to "In Progress" and assign it to yourself.
2. Create a feature branch from `develop` branch. Name it as `feature/<issue number>-<short description>`. For example, `feature/1234-add-foo`.
3. Implement the feature. Make sure all tests are passing. When push do not disable commit/push hooks otherwise (this is going to be emergency) notify other contributors about it.
4. Create a PR from the feature branch to `develop` branch.
5. Follow the steps:
    - If PR is approved, squash and rebase it into `develop` branch.
    - If PR is not approved, fix the issues and repeat step 4.
6. Delete the feature branch.
7. Move the issue into "Done" status.

### Help, I am blocked!

Sometimes, it happens.
1. Mark the issue as `Help wanted`.
2. Notify other contributors about it. Ask for help.


# Contributing Guidelines

Welcome to Hubctl. We are excited you have joined our community. This document will help you get started as a contributor.


- [Issues Handling](#issues-handling)
- [Branching Strategy](#branching-strategy)
- [Release Procedure](#release-procedure)
- [Coding Guidelines](#coding-guidelines)

## Issues Handling

All activities has been tracked as a Github issues in several repositories and have been attached to [this](https://github.com/orgs/epam/projects/8/) Github project.

### Working with Github Project

There are following statuses in the project:
- To Do: Issues that are ready to be picked up by contributors.
- In Progress: Issues that are being worked on.
- In Review: Issues that are ready to be reviewed by other contributors. Typically backed by a PR. This is a good status for PR issues.
- Done + Open: Issues that have been completed. Waiting for next released
- Done + Closed: Issues that have been released.

All issues attached to the project are in the status *"To Do"* and are ready to be picked up by contributors.

Once issue has been picked up it goes into *"In Progress"* status.

Once feature have been fully implemented it goes into *"Done"* status. and __not__ closed. Issue closed only when it has been released.

Status *"Done"* also means the feature branch have been integrated into `develop` branch (see blow about branching strategy).

Issue from *"Done"* status goes into maybe moved *"In Progress"* status (or even reopened) if additional work have been identified. We do not want to keep duplicated issues (or regressions in case of bugs) as separate issues. Instead, please re-energize the existing issue.

If duplicated issue have been identified it should be closed with labeled as `duplicated` and reference to the original issue.

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

### Before opening a new issue:

1. Please check if it has been already reported. If so, please add a comment to the existing issue.
2. If this is a regression, then reopen an existing issue.
3. If there is work in progress (corresponding issue is in progress). Then please add a comment to the existing issue or a PR

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

## Creating a PR


Before PR

1. Assign issue to yourself
2. Create a feature branch and it to the issue (Select issue => 'development' => select `Create Branch` link)
3. Implement a change.

Submitting PR:
1. PR should be created from the feature branch to `develop`.
2. Link PR to the issue (select issue => Development => press gear button => find PR) or follow guidelines on [linking by keyword](https://docs.github.com/en/issues/tracking-your-work-with-issues/linking-a-pull-request-to-an-issue#linking-a-pull-request-to-an-issue-using-a-keyword)
3. All DOD items should be completed, and obviously all tests should be passing.
4. PR should be reviewed by at least one other contributor.
5. Squash and rebase PR into `develop` branch. The squashed commit message should contain original issue reference. For example, `Update foo bar #1234`
6. Delete the feature branch.

> Note: PR doesn't have to be attached to the project

Example of PR text

```markdown
## What type of PR is this?

## What this PR does / why we need it:

## Which issue(s) this PR fixes:

Fixes #

## Special notes for your reviewer:

Does this PR introduce a user-facing change?
```

### Help, I am blocked!

Sometimes, it happens.
1. Mark the issue as [`Help wanted`](/epam/hubctl/labels/help%20wanted).
2. Notify other contributors about it. Ask for help.

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

### Commit Messages

Few guidelines how to write good commit messages. Even if commits will be squashed afterwards it will make easier to identify what has been changed and why. Here are the guidelines for good commit messages:

1. Git commit message should always have a subject line and optionally a body.
2. Git commit message should always have a reference to the issue number in a postfix
3. Git commit message should be written in a present tense. For example, "Add foo" and should start from a capital letter.
3. Git commit message should explain exactly what has been done. Such messages as: `Changes in readme` or `Make code better` are not acceptable.
4. Avoid repetitive commits messages. Same applies to rephrase the same message. For example, "Fix foo" and "Foo has been fixed" are not acceptable.
5. When you are fixing a bug and using squashing a PR: then use word "Fix foo #123" in the commit message. This will automatically close the issue once PR has been merged.

## Release Guidelines

At the moment there is no release schedule. With the current pace of the development, release has been agreed by the contributors. This means release can be (and should be) done at any random moment of time. This emphasizes the importance of continuous relative stability of `develop` branch.

### Before begin

Make sure all issues have been completed. It should not be possible to make a release for 'In Progress' issues. If there is an issue in 'In Progress' but the code has been committed to `develop` then this is blocker for release.

> For example: the feature has been implemented, yet DOD requires to update documenation published in [hubct.io](/epam/hubctl.io) that is different repository. This means  Release should not happen then.

### Start the release

1. Release is always taken from `develop` branch
2. Do not squash it but rebase to the `main` branch (or `master` if Github legacy naming is used)
3. Create a tag with a version number. For example, `v0.1.0` (not aplicable for components and stacks). This should trigger the release workflow.
4. Close issues that have been released.

## Coding Guidelines

### Changing the Shell Scripts

Hubctl has a lot of shell scripts. Extensions, Hooks and pre/post deployment scripts are basically implemented as the shell scripts. This makes it easily extendable and "hackable".

Few notes when you change the shell scripts:

1. POSIX compatible scripts and make sure you are using right shebang. Bash is not always available on the target system and should be avoided unless absolutely must.

```bash
#/bin/sh -e
```

2. Enable [shellcheck](http://shellcheck.net) in your editor. It is a great tool that helps to avoid common mistakes.

3. Do not disable shellcheck warnings on a script level unless you are absolutely sure what you are doing. Instead, disable it on a line level.

```bash
#!/bin/sh -e
#shellcheck disable=SC2039
```

If false-positive waring, then disable it in-place.

### Writing Readme for Component

Every component needs to have a README.md file. It should contain the following information:

```markdown
# Title

Text description of the component

## Requirements

List of the tools and software required to run the component

Example:
* [helm](https://helm.sh)
* [kustomize](https://kustomize.io)

## Dependencies

List of component dependencies (can be other components)

Example:
* Kubernetes cluster
* Cert Manager
* Istio

## Parameters

| Name      | Description | Default Value | Required
| --------- | ---------   | ---------     | :---: |
| `parameter.name` | Meaning for parameter | `default value` | `x`

## Implementation Details

Directory content description.

To display directory content use the following command and comments for each file

Write a deployment algorithm if applicable.

## See also

List of other components or what user should look next
```

See [kserve component](https://github.com/epam/hub-kubeflow-components/tree/main/kserve) for the example.

Do not list `.env`, `.gitignore` and `.hub` and similar files or directories. It does not add any value. If a config has bene rendered by template. Then do not list a rendered file. Template file is just enough

### Writing Readme for Stack

```markdown
# Title

description of a stack

## Requirements

List of required tools needed to deploy a stack

## Dependencies

List of stack requirements (such as cloud provider, Kubernetes cluster etc)

## Components

List of components that have been used in the stack

## Parameters

| Component(s) |   Parameter name       |    Description                 |  Default value |
|-------------|:------------------------|:---------------------------------|:-------------|
| `component` | `parameter.name`        | Description | default value or name of environment variable |

## Implementation Details

Directory content description.

## Validate Deployment

Instructions how to validate deployment has been successful

## Troubleshooting

Instructions how to troubleshoot deployment

## See Also

References to the other stacks user can be interested after deloying the current
```

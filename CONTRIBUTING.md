# Contributing Guidelines

Welcome to Hubctl. We are excited you have joined our community. This document will help you get started as a contributor.

## How to Contribute

All activities has been tracked as a Github issues in several repositories and have been attached to [this](https://github.com/orgs/epam/projects/8/views/2) Github project.

All issues attached to the project are in the status *"To Do"* and are ready to be picked up by contributors.

Once issue has been picked up it goes into *"In Progress"* status.
If issue has been blocked it should be labeled as `Help wanted`. This allows to track issues that are blocked and need help from other contributors.

Once feature have been fully implemented it goes into *"Done"* status. and __not__ closed. Issue closed only when it has been released.

Status *"Done"* also means the feature branch have been integrated into `develop` branch (see blow about branching strategy).

Issue from *"Done"* status goes into maybe moved *"In Progress"* status (or even reopened) if additional work have been identified. We do not want to keep duplicated issues (or regressions in case of bugs) as separate issues. Instead, please re-energize the existing issue.

If duplicated issue have been identified it should be closed with labeld `duplicated` and reference to the original issue.

### Branching Strategy

All code changes should be done in the separate Feature branch. Feature branch should have a number of the issue it is related to.

Every commit should have a reference to the issue it is related to (even if commits will be squashed later on). If by some reasons developer missed the reference, then they can `force push` to their branch to add the reference. Otherwise commit should be linked in the issue as the comment.

Once feature is ready for review, developer should create a Pull Request and assign it to the reviewer (better two reviewers).

We are using continuous integration approach. This means features should be integrated as fast as possible. Even if it brings some temporary instability, still the feature branches should be integrated at most by the end of the week and commits squashed into one commit. You should use rebase instead of merge workflow. We like to keep our history clean and readable.

All features are integrated into `develop` branch. This is the default integration branch where the latest greatest lives.

`develop` integrates into `master` branch during the release.

### Reporting Bugs

If you find a bug, please report it by opening an issue. Please include as much information as possible, including:

1. Steps to reproduce: please include the commands you ran and the output you received, as well as your environment (OS, shell, Golang setup etc.)
2. Actual behavior
3. Expected behavior

Such issue should be labeled as a `bug`.

### Suggesting Enhancements

If you have an idea for a new feature or an enhancement, please report it by opening an issue (labeled by `enhancement`). Please include as much information as possible, including:

1. Use case: please describe the use case you have in mind and why it is important
2. Proposed solution: please describe how you would like to see the feature implemented
3. Alternatives: please describe any alternative solutions or features you've considered, if any

> Enhancements should contain a checklist of the features that should be implemented. This checklist should be updated as the feature is implemented. See below

```markdown
- [ ] Lorem
- [ ] Ipsum
- [ ] Dolor
- [ ] Sit
```

Steps like "update documentation on [hubct.io](http://github.com/epam/hubctl.io) etc are good candidates for the checklist items.

If enhancement is rather large, then separate items in the checklist can be broken into multiple issues. Goal is not too have a one issue that will be in "In Progress" status for several weeks. Instead, we want to have a number of issues that can be implemented in a two days max.

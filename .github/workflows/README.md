# Hub CTL Workflows

## go.yaml

The workflow runs operations: `go test` and `go build` for two OS (ubuntu and macOS). It triggers on each push that has changes related to `*.go` files and on created pull requests.

## release.yaml

The workflow is used to create a new release of Hub CTL. It triggers on creation of a new tag in [semver](https://semver.org/) format (ex. `v1.0.0`).

The release is created with help of the [GoReleaser] tool. The config file is located at the root of the repository. [GoReleaser] builds artifacts and creates GitHub release with them. Also, it updates the Formula for [brew](https://brew.sh/) installation of Hub CTL. Repository with Formula located [here](https://github.com/epam/homebrew-hubctl). To update the Formula, [GoReleaser] requires [GitHub Personal Access Token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token). It should be stored as repository secret under `HOMEBREW_TAP_GITHUB_TOKEN` name.

[GoReleaser]: https://goreleaser.com/

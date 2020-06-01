I'm sorry, but there are no canonical unit tests for Hub CLI.
We test it via end-to-end test suites, that are proprietory.

I'd start with integration tests to cover:
- elaborate - assemble manifest from components and compare with reference file;
- pull - get the sources (this is currently extension, which is ok);
- deploy and undeploy that works on stubs with non-trivial components and parameters / outputs interaction, compare to reference output, compare processed templates and state file.

https://github.com/agilestacks/automation-hub-cli/issues/7

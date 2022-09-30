I'm sorry, but there are no canonical unit tests for Hub CTL.
We test it via end-to-end test suites, that are proprietory.

I'd [start](https://github.com/epam/hubctl/issues/7) with integration tests to cover:
- elaborate - assemble manifest from components and compare with reference file;
- pull - get the sources (this is currently extension, which is ok);
- deploy and undeploy that works on stubs with non-trivial components and parameters / outputs interaction, compare to reference output, compare processed templates and state file.

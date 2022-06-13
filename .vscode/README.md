`launch.json.example` contains an example debugging configuration. 

The easiest way how to debug the CLI locally (I assume the reader already has a basic understanding of SuperHub concepts) is explained below:

1. `launch.json.example` contains debugging configuration example for 3 most used hub cli commands:
    
    * `hub elaborate`
    * `hub deploy`
    * `hub undeploy`


2. To do `hub elaborate` choose any valid `hub` manifest (sometimes called `hub.yaml`) and adjust `args` of `hub stack elaborate` configuration
Make sure you also adjust the `env` variables of the stack.

3. To do `hub deploy` or `hub undeploy`, the recent `elaborate` file (produced by `hub elaborate` command is required). 
Adjust `args` section of the corresponding command to point to the right `elaborate` file.
Make sure you also adjust the `env` variables of the stack.


srcd

- parse
    - uast -l LANG -q QUERY files   // can read from stdin
    # when we call parse uast if the drivers required are missing will be automatically installed
    # outputs the corresponding json
    - lang FILES    // can read from stdin
    - drivers
        - list
        - install LANGS | -all|alpha
        - remove LANGS | -all
        - update LANGS | -all

- sql "query"
    # if sql has not been installed it will be installed automatically
    # if sql has not been started it will be started automatically

- version # version of the 

- web # opens a browser on the engine dashboard

- components
    - status    component_id | blank for all
    - start     component_id
    - stop      component_id
    - restart   component_id
    - install   component_id | -all
    - remove    component_id
    - update    component_id

components:
- sql
- parser (gRPC or REST?)

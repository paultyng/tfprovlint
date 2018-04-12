# TF Provider Linter

Lints Terraform provider source code for common issues.

```shell
# note the provider subdir!
$ tfprovlint github.com/terraform-providers/terraform-provider-aws/aws
```

## Rules

| ID | Description | Notes |
|---|---|---|
| tfprovlint001 | Do not call `d.SetId("")` in a `DeleteFunc` |  |
| tfprovlint002 | Only set attributes described in Schema | This can result in false positives due to issues in the static parsing of resource schema |
| tfprovlint003 | Use the proper type when setting an attribute |  |
| tfprovlint005 | Do not dereference a pointer value before calling `d.Set` |  |

## Current Limitations

* People can do weird stuff in code! This does not execute the provider, so won't be able to infer with 100% certainty the runtime schema unless the schema code is not very dynamic.

## TODO

* Clean up debug logging
* Allow toggling between failure vs warning on rules
* Allow a configuration file to turn on and off rules
* Make the partial parse state cleaner when dynamic schema is detected
* More rules!!
* See additional `TODO` comments [in the code](https://github.com/paultyng/tfprovlint/search?l=Go&q=TODO&type=)

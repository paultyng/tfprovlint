# TF Provider Linter

[![Build Status](https://travis-ci.org/paultyng/tfprovlint.svg?branch=master)](https://travis-ci.org/paultyng/tfprovlint)

Lints Terraform provider source code for common issues.

```shell
$ tfprovlint lint github.com/terraform-providers/terraform-provider-aws
```

## Rules

| ID | Description | Runtime | Notes |
|---|---|---|---|
| tfprovlint001 | Do not call `d.SetId("")` in a `DeleteFunc` |  |  |
| tfprovlint002 | Only set attributes described in Schema | false positives* |  |
| tfprovlint003 | Use the proper type when setting an attribute |  |  |
| tfprovlint005 | Do not dereference a pointer value before calling `d.Set` |  |  |
| tfprovlint006 | Cannot set both `Optional` and `Required` | Schema | not yet implemented |
| tfprovlint007 | Cannot set both `Required` and `Computed` | Schema | not yet implemented |
| tfprovlint008 | Must be one of `Required`, `Optional`, or `Computed` | Schema | not yet implemented, false positives* |
| tfprovlint009 | `Default` must be `nil` if `Computed` | Schema | not yet implemented |
| tfprovlint010 | `Default` cannot be set with `Required` | Schema | not yet implemented |
| tfprovlint011 | `ComputedWhen` can only be set with `Computed` | Schema | not yet implemented |
| tfprovlint012 | `ConflictsWith` cannot be set with `Required` | Schema | not yet implemented |
| tfprovlint013 | `Elem` must be set for `TypeList` or `TypeSet` | Schema | not yet implemented |
| tfprovlint014 | `Default` is not valid for `TypeList` or `TypeSet` | Schema | not yet implemented |
| tfprovlint015 | `Set` can only be set for `TypeSet` | Schema | not yet implemented |
| tfprovlint016 | `MinItems` and `MaxItems` are only supported on `TypeList` or `TypeSet` | Schema | not yet implemented |
| tfprovlint017 | `ValidateFunc` is not valid on a `Computed` only attribute | Schema | not yet implemented |
| tfprovlint018 | `DiffSuppressFunc` is not valid on a `Computed` only attribute | Schema | not yet implemented |
| tfprovlint019 | `ValidateFunc` is not valid for `TypeList` or `TypeSet` | Schema | not yet implemented |
| tfprovlint020 | Attribute `Name` may only contain lowercase alphanumeric characters & underscores (`^[a-z0-9_]+$`) | Schema | not yet implemented |
| tfprovlint021 | `Create`, `Update`, and `Delete` are not valid on a data source | Resource | not yet implemented |
| tfprovlint021 | `CustomizeDiff` is not valid on a data source | Resource | not yet implemented |
| tfprovlint022 | All non-`Computed` attributes must be `ForceNew` if `Update` is not defined in a resource | Resource | not yet implemented, false positives* |
| tfprovlint023 | `Update` is superfluous if all attributes are `ForceNew` or `Computed` w/out `Optional` in a resource | Resource | not yet implemented, false positives* |
| tfprovlint024 | `Read` must be implemented on a data source or resource | Resource | not yet implemented, false positives* |
| tfprovlint025 | `Delete` must be implemented on a resource | Resource | not yet implemented, false positives* |
| tfprovlint026 | Do not use reserved field names | Resource |  |
| tfprovlint027 | SDK version is less than 0.11 |  | not yet implemented |
| tfprovlint028 | Do not log resource data values |  | not yet implemented |
| tfprovlint029 | Resource functions should call `fmt.Errorf` instead of `errwrap.Wrapf` |  |  |

<!-- TODO: add rules from the importer's InternalValidate -->

## Current Limitations

* People can do weird stuff in code! This does not execute the provider, so won't be able to infer with 100% certainty the runtime schema unless the schema code is not very dynamic.

## TODO

* Finish switching to a full SSA implementation
* Allow toggling between failure vs warning on rules
* Allow a configuration file to turn on and off rules
* Make the partial parse state cleaner when dynamic schema is detected, allow "false positive" rules to be skipped
* More rules!!
* See additional `TODO` comments [in the code](https://github.com/paultyng/tfprovlint/search?l=Go&q=TODO&type=)

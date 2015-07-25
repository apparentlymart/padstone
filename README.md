# Padstone: a prototype of using Terraform for builds

This codebase is currently just a prototype of a new frontend on top of the Terraform core which serves
Packer-like use cases around building applications for later deployment.

It differs from standard Terraform in the following regards:

* The set of subcommands is more appropriate for a build workflow, compared to terraform's deployment workflow:
  * ``build``: create an empty state, read a config, and create all of the resources in this config before writing out
  a state file of the result. Unlike ``terraform ``apply this always starts from an empty state and thus creates new
  resources on each run.
  * ``destroy``: given an existing state file, destroy all of the resources in the state. This allows the resources from
  earlier builds to be easily destroyed when they are no longer required, improving on the capabilities of Packer today.
  * ``publish``: given an existing state file, publish it to a Terraform remote state backend so that it can be easily
  consumed by a downstream Terraform config using the ``terraform_remote_state`` resource.
* Has a new concept of a "temporary resource", which is created during the build process but destroyed once the main
resources have been created. This allows the creation of infrastructure that is used during the build but not needed
once the build is complete, like an EC2 instance to use to produce an AMI.

At present this is just intended as a vehicle for discussion, and has been referenced over in
[Terraform issue #2789](https://github.com/hashicorp/terraform/issues/2789).

For more details, see my blog post
[Padstone: Terraform for Software Builds](http://apparently.me.uk/padstone-terraform-for-deployment/).

----

If you'd like to build Padstone to give it a try, you'll need Go 1.4+ just like for Terraform itself. Since Padstone uses
Terraform's plugins you also need to also build the Terraform plugin binaries and place them in the same directory as the
``padstone`` binary. A relativey-easy way to achieve this is to set up a normal Terraform development environment and then
install Padstone into it:

```
go install github.com/apparentlymart/padstone/cmd/padstone
```

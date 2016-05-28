package padstone

import (
	"reflect"
	"testing"

	tfcfg "github.com/hashicorp/terraform/config"
)

func TestConfigParsing(t *testing.T) {
	config, err := ParseConfig([]byte(configTestConfig), "padstone.hcl")
	if err != nil {
		t.Fatalf("unexpected error parsing config: %s", err)
	}

	want := &Config{
		SourceFilename: "padstone.hcl",

		Variables: []*tfcfg.Variable{
			{
				Name:         "version",
				Default:      "dev",
				Description:  "version number",
				DeclaredType: "",
			},
		},
	}

	// Variable blocks
	{
		if !reflect.DeepEqual(config.Variables, want.Variables) {
			t.Fatalf("got variables %#v; want %#v", config, want)
		}
	}

	// Provider blocks
	{
		if got, want := len(config.Providers), 1; got != want {
			t.Fatalf("got %d providers; want %d", got, want)
		}

		if got, want := config.Providers[0].Name, "aws"; got != want {
			t.Fatalf("provider 0 named %q; want %q", got, want)
		}
		if got, want := config.Providers[0].Alias, ""; got != want {
			t.Fatalf("provider 0 alias %q; want %q", got, want)
		}
		if got, want := config.Providers[0].RawConfig.Raw["region"], "us-west-2"; got != want {
			t.Fatalf("provider 0 region %q; want %q", got, want)
		}
	}

	// Target blocks
	{
		if got, want := len(config.Targets), 3; got != want {
			t.Fatalf("got %d targets; want %d", got, want)
		}

		if got, want := config.Targets[0].Name, "ami"; got != want {
			t.Fatalf("target 0 named %q; want %q", got, want)
		}
		if got, want := config.Targets[1].Name, "ami_source_instance"; got != want {
			t.Fatalf("target 1 named %q; want %q", got, want)
		}
		if got, want := config.Targets[2].Name, "dev"; got != want {
			t.Fatalf("target 2 named %q; want %q", got, want)
		}

		{
			target := config.Targets[0]

			if got, want := len(target.Providers), 1; got != want {
				t.Fatalf("target 0 has %d providers; want %d", got, want)
			}
			if got, want := target.Providers[0].Name, "aws"; got != want {
				t.Fatalf("target 0 provider named %q; want %q", got, want)
			}
		}

		{
			target := config.Targets[1]

			if got, want := len(target.Providers), 0; got != want {
				t.Fatalf("target 1 has %d providers; want %d", got, want)
			}
		}
	}
}

const configTestConfig = `
variable "version" {
  default     = "dev"
  description = "version number"
}

provider "aws" {
  region = "us-west-2"
}

default_build_targets = ["ami"]
default_dev_targets = ["dev"]

target "ami" {
  provider "aws" {
    region = "us-east-1"
    alias = "use1"
  }

  resource "aws_ami_from_instance" "result" {
    instance_id = "${target.ami_source_instance.id}"
  }

  resource "aws_ami_copy" "result" {
    source_ami_id = "${aws_ami_from_instance.result.id}"
    source_region = "us-east-1"

    provider = "aws.usw2"
  }

  output "usw2_id" {
    value = "${aws_ami_from_instance.result.id}"
  }

  output "use1_id" {
    value = "${aws_ami_copy.result.id}"
  }
}

target "ami_source_instance" {
  module "build_support" {
    source = "./build_support"
  }

  data "aws_ami" "ubuntu" {
    id = "ami-06b94666"
  }

  resource "aws_instance" "result" {
    ami                   = "${data.aws_ami.ubuntu.id}"
    instance_type         = "m3.medium"
    subnet_id             = "${module.build_support.subnet_id}"
    vpc_security_group_id = ["${module.build_support.security_group_id}"]
  }

  output "id" {
    value = "${aws_instance.result.id}"
  }
}

target "dev" {
  data "docker_image" "ubuntu" {
    name = "ubuntu:xenial"
  }

  resource "docker_container" "app" {
    name  = "myapp-dev"
    image = "${data.docker_image.ubuntu.latest}"
  }
}
`

# Lambda Secrets Prefetch

[![license](http://img.shields.io/badge/license-apache_2.0-blue.svg?style=flat)](https://raw.githubusercontent.com/square/lambda-secrets-prefetch/master/LICENSE)
[![release](https://img.shields.io/github/release/square/lambda-secrets-prefetch.svg?style=flat)](https://github.com/square/lambda-secrets-prefetch/releases)
[![Tests](https://github.com/square/lambda-secrets-prefetch/workflows/Go/badge.svg)](https://github.com/square/lambda-secrets-prefetch/actions?query=workflow%3AGo)

Lambda extension to pre-fetch secrets from AWS SecretsManager. This tool is discussed in detail in our [blog post](https://developer.squareup.com/blog/using-lambda-extensions-to-accelerate-secrets-access).

## Features
**Load secrets before invocation:** This extension loads and caches secrets to `/tmp` before the lambda recieves an invoke, cutting down on response time.

**Fetch only what the lambda needs:** Performance is critical for lambdas, so this extension only pulls secrets specified in a configuration file that is shipped with your lambda function code.

## Install
We have put together a [Makefile](Makefile) that will build and deploy the extension into your account with an [example lambda](example-lambda/main.py) that prints stored secrets.
Before running any commands, ensure `AWS_PROFILE` and `AWS_REGION` are set in your terminal, and you have replaced the variables at the top of `Makefile`.
Make sure the role specified for `EXECUTION_ROLE` is an existing IAM role that has read access to the secrets which you specify to access.

```
export AWS_PROFILE=<profile name>
export AWS_REGION=<region to run in>
```

### config.yaml
This extension reads a `config.yaml` file from the root of the lambda function package. For example:

```yaml
SecretManagers:
- prefix: [arnprefix]
  secrets:
    - secretname: testsecrets1
    - secretname: testsecrets2
      filename: renamesecret
- prefix: "arn:aws:secretsmanager:us-east-1:9876543210:secret:"
  secrets:
    - secretname: anothersecret
SecretsHome: /tmp/secrets
```

The `SecretManagers` entry can be used to point to secrets in multiple accounts, where `prefix` in combination with `secretsname` should point to a fully qualified secret name.
`filename` can be used to store secrets with a particular file name, but is optional. The default is using `secretname` as `filename`.
`SecretsHome` is optional. The default is `/tmp/secrets`.

Run `make update-layer` to create the extension layer. This will write a file `.layer.arn.txt`, which is required to update the function.
Once the arn file is written, run `make create-function` which will create a function using the previously created extension layer.
If you make changes to the extension or layer, run `make update-layer` and `make update-function` to deploy the latest changes.
`make test` to run tests and to run, `make run`.

## Develop
We use [Go modules][1] for managing vendored dependencies. If you would like to contribute, see the [CONTRIBUTING.md](CONTRIBUTING.md) file for extra information.

[1]: https://github.com/golang/go/wiki/Modules

## License
```
Copyright 2020 Square, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```


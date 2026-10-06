# Multi-cluster EKS authentication with AWS profiles

BloodHound Kube delegates kubeconfig authentication to Kubernetes `client-go`.
For EKS, this normally means executing `aws eks get-token` as configured in the
kubeconfig's user entry. The AWS CLI resolves the AWS credentials and returns a
Kubernetes token; BloodHound Kube does not load or switch AWS profiles itself.

To collect from clusters using different AWS profiles in a single run, use a
separate kubeconfig file for each cluster and pin its profile in the exec
authentication configuration.

## Set up per-cluster kubeconfigs

Ensure the AWS CLI is installed and the named profiles are configured. For SSO
profiles, log in before collection:

```bash
aws sso login --profile prod-profile
aws sso login --profile staging-profile
```

Generate separate kubeconfigs (the parent directory must exist):

```bash
mkdir -p ~/.kube

aws eks update-kubeconfig \
  --name prod \
  --region us-east-1 \
  --profile prod-profile \
  --kubeconfig ~/.kube/eks-prod

aws eks update-kubeconfig \
  --name staging \
  --region us-east-1 \
  --profile staging-profile \
  --kubeconfig ~/.kube/eks-staging
```

`aws eks update-kubeconfig` sets the generated cluster as the file's current
context. When invoked with `--profile`, the AWS CLI records `AWS_PROFILE` in the
generated exec environment. Check that each file selects the intended context
and profile. For example, the prod kubeconfig's user entry should contain an
exec configuration like this (the user name must match its context's user):

```yaml
users:
  - name: eks-prod
    user:
      exec:
        apiVersion: client.authentication.k8s.io/v1beta1
        command: aws
        args:
          - eks
          - get-token
          - --region
          - us-east-1
          - --cluster-name
          - prod
          - --output
          - json
        env:
          - name: AWS_PROFILE
            value: prod-profile
```

The staging kubeconfig should reference the staging cluster and
`staging-profile`. If an existing kubeconfig uses an explicit AWS CLI `--profile`
argument instead, that also selects the profile for its authentication command.

## Configure and run collection

Create `clusters.yaml`:

```yaml
defaults:
  scope: core
  clusterConcurrency: 2
  outputDir: ./output

clusters:
  - name: prod
    kubeconfig: ~/.kube/eks-prod

  - name: staging
    kubeconfig: ~/.kube/eks-staging
```

Run collection:

```bash
./bloodhound-kube collect --clusters-config clusters.yaml
```

Each cluster's authentication command uses its own configured profile, including
when cluster pipelines run concurrently. Exec authentication allows `client-go`
to obtain fresh tokens as needed rather than relying on a token copied into the
multi-cluster YAML.

## Profile selection and runtime requirements

- **Cluster selection:** the multi-cluster YAML has no `context` field. Each
  kubeconfig's current context selects its cluster and user. The entry's `name`
  labels the collection and output; it does not select a kubeconfig context.
- **Profile selection:** the multi-cluster YAML has no `awsProfile` field. Unknown
  YAML fields are currently ignored, so adding that key will not select a
  profile. Configure the profile in the kubeconfig's exec environment or args.
- **Inherited environment:** exec commands inherit the tool's environment, with
  kubeconfig exec environment values applied to the child process. A shell-level
  `AWS_PROFILE` is shared by all commands without their own profile selection.
  Pin profiles per kubeconfig for multi-profile runs. Normal AWS credential
  precedence still applies; inherited `AWS_ACCESS_KEY_ID`,
  `AWS_SECRET_ACCESS_KEY`, and `AWS_SESSION_TOKEN` can take precedence over an
  environment-selected profile.
- **SSO sessions:** collection does not initiate an SSO login. The required
  profiles must have valid sessions before running; log in again if they expire.
- **Execution environment:** `aws` must be available on `PATH`, and the profile
  configuration, credentials, and any SSO cache must be accessible to the user
  running BloodHound Kube. This also applies when running inside a container.
- **Permissions:** the IAM identity resolved by the AWS CLI needs EKS cluster
  access and the Kubernetes permissions required for collection. A successful
  AWS login alone does not grant Kubernetes access.

See the [multi-cluster collection documentation](../README.md#multi-cluster-collection)
and [annotated configuration example](../clusters.example.yml) for other settings.

# 🍹️ Refreshment

A brisk, pleasant tool to generate and use new sts tokens with an MFA token.

This automates the process of using
[AWS MFA credentials](https://aws.amazon.com/premiumsupport/knowledge-center/authenticate-mfa-cli/).

## Usage

In `~/.aws/credentials` we care about two sets of credentials. There's the `[default]` which gets updated by this tool, and a `[base]` which actually has permissions to request new credentials from `sts`.

```sh
AWS_PROFILE=base refreshment -m "arn:aws:iam::${YOUR_AWS_ACCOUNT_ID}:mfa/${AWS_MFA_NAME}" -t "${MFA_TOKEN}"
```

The tool will read your `.aws/credentials`, using the ones under the `[base]` profile to submit an API request to Amazon, and will write the values into your `default` profile.

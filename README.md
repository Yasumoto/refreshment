# 🍹️ Refreshment

A brisk, pleasant tool to generate and use new sts tokens with an MFA token.

This automates the process of using
[AWS MFA credentials](https://aws.amazon.com/premiumsupport/knowledge-center/authenticate-mfa-cli/).

## Usage

In `~/.aws/credentials` we care about two sets of credentials. There's the `[default]` which gets updated by this tool, and a `[base]` which actually has permissions to request new credentials from `sts`.

Here's an example of what your `~/.aws/credentials` should look like:

```sh
❯ cat ~/.aws/credentials                                                                                                                                  
[default]
aws_access_key_id     = this will be updated by refreshment, and is the profile that will be used to make normal aws requests
aws_secret_access_key = this will be updated by refreshment
aws_session_token     = this will be updated by refreshment

[base]
aws_access_key_id     = SET_THIS_!!!!_REPLACE_WITH_THE_ACCESS_KEY_ID_YOU_GOT_FROM_AWS
aws_secret_access_key = SET_THIS_!!!!_REPLACE_WITH_YOUR_SECRET_KEY_GENERATED_BY_AWS
```

Then to use the tool, run this (setting the `YOUR_AWS_ACCOUNT_ID`, `AWS_MFA_NAME`, and `MFA_TOKEN` variables, of course):

```sh
AWS_PROFILE=base refreshment -m "arn:aws:iam::${YOUR_AWS_ACCOUNT_ID}:mfa/${AWS_MFA_NAME}" -t "${MFA_TOKEN}"
```

The tool will read your `.aws/credentials`, using the ones under the `[base]` profile to submit an API request to Amazon, and will write the values into your `default` profile.

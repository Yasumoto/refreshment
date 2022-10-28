/*
Copyright © 2021 Joe Smith <yasumoto7@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/mitchellh/go-homedir"

	"github.com/spf13/cobra"

	"github.com/spf13/viper"

	"gopkg.in/ini.v1"
)

var cfgFile string
var token string
var mfaSerial string
var pathToSubstrate string
var terraformRootPath string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "refreshment",
	Short: "Update your AWS credentials with your MFA token",
	Long: `Refreshment is a pleasant tool to automatically
renew your AWS credentials with a new token from your multifactor
auth device. It will update your default profile in
~/.aws/credentials.`,

	Run: func(cmd *cobra.Command, args []string) {
		l := log.New(os.Stderr, "", 0)
		var profile_name string
		if mfaSerial == "" {
			l.Println("Using Substrate-based creds")
			profile_name = "refreshment_substrate"
		} else {
			l.Println("Using MFA-based creds")
			profile_name = "refreshment_mfa"
		}
		credential_file_path, err := homedir.Expand("~/.aws/credentials")
		if err != nil {
			l.Printf("Could not read homedir: %v", err)
			os.Exit(1)
		}

		cfg, err := ini.Load(credential_file_path)
		if err != nil {
			l.Printf("Fail to read aws credentials file: %v", err)
			os.Exit(1)
		}

		// Do we already have valid credentials we can swap to?
		section := cfg.Section(profile_name)
		access_key_id := section.Key("aws_access_key_id").String()
		secret_access_key := section.Key("aws_secret_access_key").String()
		session_token := section.Key("aws_session_token").String()
		if section == nil || access_key_id == "" || secret_access_key == "" || session_token == "" {
			l.Println("Credentials values are empty, generating new creds!")
		} else {
			l.Printf("Found configuration for %s\n", profile_name)
			existing_credentials := credentials.NewStaticCredentials(
				access_key_id,
				secret_access_key, session_token,
			)

			existing_service := sts.New(session.Must(session.NewSession(&aws.Config{
				Credentials: existing_credentials})))
			get_caller_identity_input := &sts.GetCallerIdentityInput{}
			_, err := existing_service.GetCallerIdentity(get_caller_identity_input)
			if err == nil {
				l.Println("Existing credentials are valid, updating default!")
				section_name := "default"
				if profile_name == "refreshment_mfa" {
					section_name = "nlk_corp"
				}
				cfg.Section(section_name).Key("aws_access_key_id").SetValue(access_key_id)
				cfg.Section(section_name).Key("aws_secret_access_key").SetValue(secret_access_key)
				cfg.Section(section_name).Key("aws_session_token").SetValue(session_token)

				cfg.SaveTo(credential_file_path)

				l.Println("Swapped in existing credentials, rock n' roll 🎸")

				return
			}
		}

		// https://github.com/aws/aws-sdk-go/blob/main/service/sts/examples_test.go#L328
		svc := sts.New(session.New())

		// 2FA-based auth
		if mfaSerial != "" {
			input := &sts.GetSessionTokenInput{
				DurationSeconds: aws.Int64(129600),
				SerialNumber:    aws.String(mfaSerial),
				TokenCode:       aws.String(token),
			}

			result, err := svc.GetSessionToken(input)

			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					switch aerr.Code() {
					case sts.ErrCodeRegionDisabledException:
						l.Println(sts.ErrCodeRegionDisabledException, aerr.Error())
					default:
						l.Println(aerr.Error())
					}
				} else {
					// Print the error, cast err to awserr.Error to get the Code and
					// Message from an error.
					l.Println(err.Error())
				}
				return
			}
			for _, profile_name := range []string{"nlk_corp", "refreshment_mfa"} {
				cfg.Section(profile_name).Key("aws_access_key_id").SetValue(*result.Credentials.AccessKeyId)
				cfg.Section(profile_name).Key("aws_secret_access_key").SetValue(*result.Credentials.SecretAccessKey)
				cfg.Section(profile_name).Key("aws_session_token").SetValue(*result.Credentials.SessionToken)
			}
			cfg.SaveTo(credential_file_path)
		} else {
			if pathToSubstrate == "" {
				l.Println("Error: Please pass the location of the Substrate binary!")
				return
			}
			if terraformRootPath == "" {
				l.Println("Error: Please pass the location of your root Substrate directory!")
				return
			}
			// Substrate-based auth
			l.Printf("Invoking %s\n", pathToSubstrate)
			substrate := exec.Command(pathToSubstrate, "credentials")
			substrate.Dir = terraformRootPath
			substrate.Stderr = os.Stderr
			substrate.Stdout = os.Stdout
			substrate.Stdin = os.Stdin
			if err != nil {
				l.Printf("%v\n", err)
				return
			}
			substrate.Run()
			// TODO(joe): Implement the rest of this here, in the meantime call via a fish shell wrapper in joe-dotfiles
			// We want to write to stdout but still view the output so we can parse it
		}

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.refreshment.yaml)")

	rootCmd.Flags().StringVarP(&mfaSerial, "mfaSerial", "m", "", "Serial Number (ARN) of your MFA device")
	rootCmd.Flags().StringVarP(&token, "token", "t", "", "Generated token from your MFA device")
	rootCmd.Flags().StringVarP(&pathToSubstrate, "pathToSubstrate", "p", "", "Location of your substrate binary")
	rootCmd.Flags().StringVarP(&terraformRootPath, "terraformRootPath", "r", "", "Location of your substrate root (containing your modules/ and root-modules/)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".refreshment" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".refreshment")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

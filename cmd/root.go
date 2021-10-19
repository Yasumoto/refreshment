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
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "refreshment",
	Short: "Update your AWS credentials with your MFA token",
	Long: `Refreshment is a pleasant tool to automatically
renew your AWS credentials with a new token from your multifactor
auth device. It will update your default profile in
~/.aws/credentials.`,

	Run: func(cmd *cobra.Command, args []string) {
		credential_file_path, err := homedir.Expand("~/.aws/credentials")
		if err != nil {
			fmt.Printf("Could not read homedir: %v", err)
			os.Exit(1)
		}

		cfg, err := ini.Load(credential_file_path)
		if err != nil {
			fmt.Printf("Fail to read aws credentials file: %v", err)
			os.Exit(1)
		}

		// https://github.com/aws/aws-sdk-go/blob/main/service/sts/examples_test.go#L328
		svc := sts.New(session.New())

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
					fmt.Println(sts.ErrCodeRegionDisabledException, aerr.Error())
				default:
					fmt.Println(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				fmt.Println(err.Error())
			}
			return
		}

		cfg.Section("default").Key("aws_access_key_id").SetValue(*result.Credentials.AccessKeyId)
		cfg.Section("default").Key("aws_secret_access_key").SetValue(*result.Credentials.SecretAccessKey)
		cfg.Section("default").Key("aws_session_token").SetValue(*result.Credentials.SessionToken)

		cfg.SaveTo(credential_file_path)
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
	rootCmd.MarkFlagRequired("mfaSerial")

	rootCmd.Flags().StringVarP(&token, "token", "t", "", "Generated token from your MFA device")
	rootCmd.MarkFlagRequired("token")
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

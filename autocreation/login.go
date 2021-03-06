// Copyright (c) 2016 Spinpunch, Inc. All Rights Reserved.
// See License.txt for license information.

package autocreation

import (
	"github.com/mattermost/mattermost-load-test/loadtestconfig"
	"github.com/mattermost/platform/model"
)

type LoginUsersResult struct {
	SessionTokens []string
	Errors        []error
}

func LoginUsers(client *model.Client, config *loadtestconfig.UsersConfiguration, users []string) *LoginUsersResult {
	loginResults := &LoginUsersResult{
		SessionTokens: make([]string, 0, len(users)),
		Errors:        make([]error, 0, len(users)),
	}

	client.Logout()

	for _, userId := range users {
		_, err := client.LoginById(userId, config.UserPassword)
		if err != nil {
			loginResults.Errors = append(loginResults.Errors, err)
		} else {
			loginResults.SessionTokens = append(loginResults.SessionTokens, client.AuthToken)
		}
	}

	return loginResults
}

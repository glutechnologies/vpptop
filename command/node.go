/*
 * Copyright (c) 2019 PANTHEON.tech.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package command

import (
	"fmt"
	"net"
	"os"

	"github.com/spf13/cobra"
)

var nodeCmd = &cobra.Command{
	Use:   "node <ip-address>",
	Short: "Collect VPP statistics from a remote IP address",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		address, err := remoteNodeAddress(args[0])
		if err != nil {
			return err
		}

		logs, err := os.Create("remote.log")
		if err != nil {
			return fmt.Errorf("error occurred while creating file: %v", err)
		}
		defer logs.Close()

		return startClient("", address, logs)
	},
}

func remoteNodeAddress(value string) (string, error) {
	ip := net.ParseIP(value)
	if ip == nil {
		return "", fmt.Errorf("invalid node IP address %q", value)
	}
	return net.JoinHostPort(ip.String(), "7878"), nil
}

func init() {
	rootCmd.AddCommand(nodeCmd)
}

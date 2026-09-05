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
	"io"
	"log"
	"os"

	"github.com/glutechnologies/vpptop/client"
	"github.com/glutechnologies/vpptop/gui"
)

// startClient is a blocking call that starts
// the terminal frontend for displaying VPP metrics.
func startClient(socket, rAddr string, logFile io.Writer) error {
	var lightTheme bool
	if _, lightTheme = os.LookupEnv("VPPTOP_THEME_LIGHT"); lightTheme {
		gui.SetLightTheme()
	}

	log.SetOutput(logFile)
	app, err := client.NewApp(lightTheme, logFile)
	if err != nil {
		return fmt.Errorf("error occurred during client init: %v", err)
	}
	if err = app.Init(socket, rAddr); err != nil {
		return fmt.Errorf("error occurred during client init: %v", err)
	}

	app.Run()
	return nil
}

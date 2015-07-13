/*
 * Minio Client, (C) 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"os"
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

// Help message.
var sessionCmd = cli.Command{
	Name:   "session",
	Usage:  "Manage sessions for cp and sync",
	Action: runSessionCmd,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} [SESSION] {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. List sessions
      $ mc {{.Name}} list

   2. Resume session
      $ mc {{.Name}} resume [SESSION]

   3. Clear session
      $ mc {{.Name}} clear [SESSION]|[all]

`,
}

func listSessions() error {
	for _, sid := range getSessionIDs() {
		s, err := loadSessionV2(sid)
		if err != nil {
			return iodine.New(err, nil)
		}
		console.Prints(s)
	}
	return nil
}

func clearSession(sid string) {
	if sid == "all" {
		for _, sid := range getSessionIDs() {
			session, err := loadSessionV2(sid)
			if err != nil {
				console.Fatalf("Unable to load session ‘%s’, %s", sid, iodine.New(err, nil))
			}
			session.Close()
		}
		return
	}

	if !isSession(sid) {
		console.Fatalf("Session ‘%s’ not found.\n", sid)
	}

	session, err := loadSessionV2(sid)
	if err != nil {
		console.Fatalf("Unable to load session ‘%s’, %s", sid, iodine.New(err, nil))
	}
	session.Close()
}

func sessionExecute(s *sessionV2) {
	switch s.Header.CommandType {
	/*
				case "cp":
					for cps := range doCopyCmdSession(bar, s) {
		  				if cps.Error != nil {
							console.Errors(ErrorMessage{
								Message: "Failed with",
								Error:   iodine.New(cps.Error, nil),
							})
						}
						if cps.Done {
							if err := saveSessionV2(s); err != nil {
								console.Fatals(ErrorMessage{
									Message: "Failed with",
									Error:   iodine.New(err, nil),
								})
							}
							console.Println()
							console.Infos(InfoMessage{
								Message: "Session terminated. To resume session type ‘mc session resume " + s.SessionID + "’",
							})
							// this os.Exit is needed really to exit in-case of "os.Interrupt"
							os.Exit(0)
						}
					}
	*/
	case "sync":
		doSyncCmdSession(s)
	}
}

func runSessionCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 1 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
	}
	if strings.TrimSpace(ctx.Args().First()) == "" {
		cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
	}
	if !isSessionDirExists() {
		if err := createSessionDir(); err != nil {
			console.Fatals(ErrorMessage{
				Message: "Failed with",
				Error:   iodine.New(err, nil),
			})
		}
	}
	switch strings.TrimSpace(ctx.Args().First()) {
	// list resumable sessions
	case "list":
		err := listSessions()
		if err != nil {
			console.Fatals(ErrorMessage{
				Message: "Failed with",
				Error:   iodine.New(err, nil),
			})
		}
	case "resume":
		if len(ctx.Args().Tail()) != 1 {
			cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
		}
		if strings.TrimSpace(ctx.Args().Tail().First()) == "" {
			cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
		}

		sid := strings.TrimSpace(ctx.Args().Tail().First())

		_, err := os.Stat(getSessionFile(sid))
		if err != nil {
			console.Fatals(ErrorMessage{
				Message: "Failed with",
				Error:   iodine.New(errInvalidSessionID{id: sid}, nil),
			})
		}

		s, err := loadSessionV2(sid)
		if err != nil {
			console.Fatals(ErrorMessage{
				Message: "Failed with",
				Error:   iodine.New(errInvalidSessionID{id: sid}, nil),
			})
		}

		savedCwd, err := os.Getwd()
		if err != nil {
			console.Fatals(ErrorMessage{
				Message: "Failed with",
				Error:   iodine.New(err, nil),
			})
		}
		if s.Header.RootPath != "" {
			// chdir to RootPath
			os.Chdir(s.Header.RootPath)
		}

		sessionExecute(s)
		err = s.Close()
		if err != nil {
			console.Fatals(ErrorMessage{
				Message: "Failed with",
				Error:   iodine.New(err, nil),
			})
		}

		// change dir back
		os.Chdir(savedCwd)

	// purge a requested pending session, if "*" purge everything
	case "clear":
		if len(ctx.Args().Tail()) != 1 {
			cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
		}
		if strings.TrimSpace(ctx.Args().Tail().First()) == "" {
			cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
		}
		clearSession(strings.TrimSpace(ctx.Args().Tail().First()))
	default:
		cli.ShowCommandHelpAndExit(ctx, "session", 1) // last argument is exit code
	}
}
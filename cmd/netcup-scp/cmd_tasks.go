// Copyright 2026 Laurynas Četyrkinas <laurynas@digilol.net>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/digilolnet/go-netcup-scp/pkg/scp"
)

func newTasksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tasks",
		Short: "Manage async tasks",
	}
	cmd.AddCommand(
		newTasksListCmd(),
		newTasksGetCmd(),
		newTasksCancelCmd(),
	)
	return cmd
}

func newTasksListCmd() *cobra.Command {
	var serverID int
	var state, q string
	var limit, offset int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List async tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			opts := &scp.ListTasksOptions{}
			if serverID > 0 {
				opts.ServerId = new(int32(serverID))
			}
			if state != "" {
				s := scp.TaskState(state)
				opts.State = &s
			}
			if q != "" {
				opts.Q = &q
			}
			if limit > 0 {
				opts.Limit = new(int32(limit))
			}
			if offset > 0 {
				opts.Offset = new(int32(offset))
			}

			tasks, err := cc.client.ListTasks(cc.ctx, opts)
			if err != nil {
				return err
			}
			return printResult(cc, tasks, func() {
				tbl := newTable("UUID", "STATE", "NAME", "STARTED (UTC)", "FINISHED (UTC)")
				for _, t := range tasks {
					tbl.AppendRow(table.Row{derefStr(t.Uuid), derefStr((*string)(t.State)), derefStr(t.Name), fmtTime(t.StartedAt), fmtTime(t.FinishedAt)})
				}
				tbl.Render()
			})
		},
	}
	cmd.Flags().IntVar(&serverID, "server-id", 0, "filter by server ID")
	cmd.Flags().StringVar(&state, "state", "", "filter by state (QUEUED, RUNNING, DONE, FAILED, CANCELLED)")
	cmd.Flags().StringVar(&q, "q", "", "search query")
	cmd.Flags().IntVar(&limit, "limit", 0, "max results")
	cmd.Flags().IntVar(&offset, "offset", 0, "pagination offset")
	return cmd
}

func newTasksGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "get <uuid>",
		Short:             "Get task details",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(taskUUIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			task, err := cc.client.GetTask(cc.ctx, args[0])
			if err != nil {
				return err
			}
			return printResult(cc, task, func() {
				printKV(
					"UUID", derefStr(task.Uuid),
					"Name", derefStr(task.Name),
					"State", derefStr((*string)(task.State)),
					"Message", derefStr(task.Message),
					"Started At (UTC)", fmtTime(task.StartedAt),
					"Finished At (UTC)", fmtTime(task.FinishedAt),
				)
			})
		},
	}
}

func newTasksCancelCmd() *cobra.Command {
	var wait bool
	cmd := &cobra.Command{
		Use:               "cancel <uuid>",
		Short:             "Cancel a running task",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(taskUUIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			task, err := cc.client.CancelTask(cc.ctx, args[0])
			if err != nil {
				return err
			}
			return printTaskAndWait(cc, task, wait)
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "wait for task to reach terminal state")
	return cmd
}

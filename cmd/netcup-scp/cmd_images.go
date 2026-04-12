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
	"fmt"
	"os"
	"path/filepath"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func newImagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "images",
		Short: "Manage user-uploaded disk images",
	}
	cmd.AddCommand(
		newImagesListCmd(),
		newImagesGetCmd(),
		newImagesDeleteCmd(),
		newImagesUploadCmd(),
	)
	return cmd
}

func newImagesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List your uploaded disk images",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			images, err := cc.client.ListUserImages(cc.ctx, cc.userID)
			if err != nil {
				return err
			}
			return printResult(cc, images, func() {
				t := newTable("KEY", "SIZE (bytes)", "LAST MODIFIED (UTC)")
				for _, img := range images {
					t.AppendRow(table.Row{derefStr(img.Key), deref(img.SizeInB), fmtTime(img.LastModified)})
				}
				t.Render()
			})
		},
	}
}

func newImagesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "get <key>",
		Short:             "Get image download info",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(imageKeyCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			info, err := cc.client.GetUserImage(cc.ctx, cc.userID, args[0])
			if err != nil {
				return err
			}
			return printResult(cc, info, func() {
				printKV(
					"Filename", derefStr(info.Filename),
					"Presigned URL", derefStr(info.PresignedUrl),
					"Valid (hours)", derefInt32(info.PresignedUrlValidityDurationInHours),
				)
			})
		},
	}
}

func newImagesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "delete <key>",
		Short:             "Delete a user-uploaded image",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(imageKeyCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			if err := cc.client.DeleteUserImage(cc.ctx, cc.userID, args[0]); err != nil {
				return err
			}
			printOK(cc)
			return nil
		},
	}
}

func newImagesUploadCmd() *cobra.Command {
	var key string
	cmd := &cobra.Command{
		Use:   "upload <file>",
		Short: "Upload an image file",
		Long: `Upload an image file to your account's object storage.

The key is the name under which the image is stored (e.g. "debian-12.qcow2").
It is used to reference the image in other commands such as "get" and "delete",
and when installing the image onto a server. Defaults to the file's basename.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			filePath := args[0]
			if key == "" {
				key = filepath.Base(filePath)
			}

			f, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("open file: %w", err)
			}
			defer f.Close()
			if err := cc.client.UploadImage(cc.ctx, cc.userID, key, f); err != nil {
				return err
			}
			printOK(cc)
			return nil
		},
	}
	cmd.Flags().StringVar(&key, "key", "", "storage key (default: filename)")
	return cmd
}

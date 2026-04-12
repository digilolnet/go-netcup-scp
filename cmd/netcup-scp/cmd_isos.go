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

	"github.com/digilolnet/go-netcup-scp/pkg/scp"
)

const (
	// autoMultipartThreshold is the file size above which multipart upload is
	// used automatically (100 MiB).
	autoMultipartThreshold = 100 * 1024 * 1024
	// defaultPartSizeMiB is the default multipart part size in MiB (50 MiB).
	defaultPartSizeMiB = 50
)

func newIsosCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "isos",
		Short: "Manage ISO images attached to a server",
	}
	cmd.AddCommand(
		newIsosListAvailableCmd(),
		newIsosGetAttachedCmd(),
		newIsosAttachCmd(),
		newIsosDetachCmd(),
	)
	return cmd
}

func newIsosListAvailableCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "list <server-id>",
		Short:             "List available ISO images for a server",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := parseID(args[0], "server-id")
			if err != nil {
				return err
			}
			isos, err := cc.client.ListAvailableISOs(cc.ctx, id)
			if err != nil {
				return err
			}
			return printResult(cc, isos, func() {
				t := newTable("ID", "ARCHITECTURE", "NAME", "DESCRIPTION")
				for _, iso := range isos {
					t.AppendRow(table.Row{derefInt32(iso.Id), string(iso.Architecture), iso.Name, derefStr(iso.Description)})
				}
				t.Render()
			})
		},
	}
}

func newIsosGetAttachedCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "get <server-id>",
		Short:             "Get the currently attached ISO",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := parseID(args[0], "server-id")
			if err != nil {
				return err
			}
			iso, err := cc.client.GetAttachedISO(cc.ctx, id)
			if err != nil {
				return err
			}
			return printResult(cc, iso, func() {
				printKV("ISO", derefStr(iso.Iso), "Attached", deref(iso.IsoAttached))
			})
		},
	}
}

func newIsosAttachCmd() *cobra.Command {
	var isoID int
	var userIso string
	var bootCdrom bool
	cmd := &cobra.Command{
		Use:               "attach <server-id>",
		Short:             "Attach an ISO to a server",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := parseID(args[0], "server-id")
			if err != nil {
				return err
			}
			opts := &scp.AttachISOOptions{}
			if isoID > 0 {
				opts.IsoID = ptr(int32(isoID))
			}
			if userIso != "" {
				opts.UserIsoName = &userIso
			}
			if bootCdrom {
				opts.ChangeBootDeviceToCdrom = ptr(true)
			}
			if err := cc.client.AttachISO(cc.ctx, id, opts); err != nil {
				return err
			}
			printOK(cc)
			return nil
		},
	}
	cmd.Flags().IntVar(&isoID, "iso-id", 0, "ID of a standard ISO image")
	cmd.Flags().StringVar(&userIso, "user-iso", "", "name of a user-uploaded ISO")
	cmd.Flags().BoolVar(&bootCdrom, "boot-cdrom", false, "change boot device to CDROM after attaching")
	return cmd
}

func newIsosDetachCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "detach <server-id>",
		Short:             "Detach the current ISO from a server",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(serverIDCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			id, err := parseID(args[0], "server-id")
			if err != nil {
				return err
			}
			if err := cc.client.DetachISO(cc.ctx, id); err != nil {
				return err
			}
			printOK(cc)
			return nil
		},
	}
}

// --- user-isos ---

func newUserIsosCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user-isos",
		Short: "Manage user-uploaded ISO images",
	}
	cmd.AddCommand(
		newUserIsosListCmd(),
		newUserIsosGetCmd(),
		newUserIsosDeleteCmd(),
		newUserIsosUploadCmd(),
		newUserIsosUploadURLCmd(),
	)
	return cmd
}

func newUserIsosListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List your uploaded ISOs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			isos, err := cc.client.ListUserISOs(cc.ctx, cc.userID)
			if err != nil {
				return err
			}
			return printResult(cc, isos, func() {
				t := newTable("KEY", "SIZE (bytes)", "LAST MODIFIED (UTC)")
				for _, iso := range isos {
					t.AppendRow(table.Row{derefStr(iso.Key), deref(iso.SizeInB), fmtTime(iso.LastModified)})
				}
				t.Render()
			})
		},
	}
}

func newUserIsosGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "get <key>",
		Short:             "Get user ISO download info",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(userISOKeyCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			info, err := cc.client.GetUserISO(cc.ctx, cc.userID, args[0])
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

func newUserIsosDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "delete <key>",
		Short:             "Delete a user-uploaded ISO",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: makeCompleter(userISOKeyCompletions),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			if err := cc.client.DeleteUserISO(cc.ctx, cc.userID, args[0]); err != nil {
				return err
			}
			printOK(cc)
			return nil
		},
	}
}

func newUserIsosUploadURLCmd() *cobra.Command {
	var key string
	cmd := &cobra.Command{
		Use:   "upload-url <file>",
		Short: "Get a presigned PUT URL for uploading an ISO",
		Long: `Get a presigned PUT URL for uploading an ISO image.

The key is the name under which the ISO will be stored (e.g. "debian-12.iso").
It is used to reference the ISO in other commands such as "get" and "delete",
and when attaching it to a server. Defaults to the file's basename.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, cleanup, err := makeCmdContext(false)
			if err != nil {
				return err
			}
			defer cleanup()

			if key == "" {
				key = filepath.Base(args[0])
			}
			url, err := cc.client.InitiateISOUpload(cc.ctx, cc.userID, key)
			if err != nil {
				return err
			}
			return printResult(cc, map[string]string{"url": url}, func() {
				fmt.Println(url)
			})
		},
	}
	cmd.Flags().StringVar(&key, "key", "", "storage key (default: filename)")
	return cmd
}

func newUserIsosUploadCmd() *cobra.Command {
	var key string
	var forceMultipart bool
	var partSizeMiB int
	cmd := &cobra.Command{
		Use:   "upload <file>",
		Short: "Upload an ISO file",
		Long: `Upload an ISO file to your account's object storage.

The key is the name under which the ISO will be stored (e.g. "debian-12.iso").
It is used to reference the ISO in other commands such as "get" and "delete",
and when attaching it to a server. Defaults to the file's basename.`,
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

			fi, err := f.Stat()
			if err != nil {
				return fmt.Errorf("stat file: %w", err)
			}
			fileSize := fi.Size()
			partSize := int64(partSizeMiB) * 1024 * 1024

			useMultipart := forceMultipart || fileSize > autoMultipartThreshold

			fmt.Fprintf(os.Stderr, "Uploading %s (%s)", key, formatBytes(fileSize))
			if useMultipart {
				totalParts := (fileSize + partSize - 1) / partSize
				fmt.Fprintf(os.Stderr, " via multipart (%d parts of %s)\n", totalParts, formatBytes(partSize))

				progress := func(partNum int, done, total int64) {
					printMultipartProgress(partNum, done, total)
				}
				if err := cc.client.UploadISOMultipart(cc.ctx, cc.userID, key, f, fileSize, partSize, progress); err != nil {
					fmt.Fprintln(os.Stderr)
					return err
				}
			} else {
				fmt.Fprintln(os.Stderr)
				pr := &progressReader{
					r:     f,
					total: fileSize,
					onRead: func(done, total int64) {
						printUploadProgress(done, total)
					},
				}
				if err := cc.client.UploadISO(cc.ctx, cc.userID, key, pr, fileSize); err != nil {
					fmt.Fprintln(os.Stderr)
					return err
				}
			}

			fmt.Fprintf(os.Stderr, "\r  done (%s)          \n", formatBytes(fileSize))
			printOK(cc)
			return nil
		},
	}
	cmd.Flags().StringVar(&key, "key", "", "storage key (default: filename)")
	cmd.Flags().BoolVar(&forceMultipart, "multipart", false, "force multipart upload regardless of file size")
	cmd.Flags().IntVar(&partSizeMiB, "part-size", defaultPartSizeMiB, "part size in MiB for multipart upload")
	return cmd
}

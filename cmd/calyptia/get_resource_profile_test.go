package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	cloud "github.com/calyptia/api/types"
)

func Test_newCmdGetResourceProfiles(t *testing.T) {
	t.Run("no_arg", func(t *testing.T) {
		got := &bytes.Buffer{}
		cmd := newCmdGetResourceProfiles(configWithMock(nil))
		cmd.SetOutput(got)

		err := cmd.Execute()
		wantErrMsg(t, `required flag(s) "aggregator" not set`, err)
	})

	t.Run("empty", func(t *testing.T) {
		got := &bytes.Buffer{}
		cmd := newCmdGetResourceProfiles(configWithMock(nil))
		cmd.SetOutput(got)
		cmd.SetArgs([]string{"--aggregator=" + zeroUUID4})

		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, "NAME STORAGE-MAX-CHUNKS-UP STORAGE-SYNC-FULL STORAGE-BACKLOG-MEM-LIMIT STORAGE-VOLUME-SIZE STORAGE-MAX-CHUNKS-PAUSE CPU-BUFFER-WORKERS CPU-LIMIT CPU-REQUEST MEM-LIMIT MEM-REQUEST AGE\n", got.String())

		t.Run("with_ids", func(t *testing.T) {
			got.Reset()
			cmd.SetArgs([]string{"--aggregator=" + zeroUUID4, "--show-ids"})

			err := cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, "ID NAME STORAGE-MAX-CHUNKS-UP STORAGE-SYNC-FULL STORAGE-BACKLOG-MEM-LIMIT STORAGE-VOLUME-SIZE STORAGE-MAX-CHUNKS-PAUSE CPU-BUFFER-WORKERS CPU-LIMIT CPU-REQUEST MEM-LIMIT MEM-REQUEST AGE\n", got.String())
		})
	})

	t.Run("error", func(t *testing.T) {
		want := errors.New("internal error")
		cmd := newCmdGetResourceProfiles(configWithMock(&ClientMock{
			ResourceProfilesFunc: func(ctx context.Context, aggregatorID string, params cloud.ResourceProfilesParams) ([]cloud.ResourceProfile, error) {
				return nil, want
			},
		}))
		cmd.SetArgs([]string{"--aggregator=" + zeroUUID4})
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true

		got := cmd.Execute()
		wantEq(t, want, got)
	})

	t.Run("ok", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		want := []cloud.ResourceProfile{{
			ID:                     "resource_profile_id_1",
			Name:                   "name_1",
			StorageMaxChunksUp:     1,
			StorageSyncFull:        true,
			StorageBacklogMemLimit: "2Mib",
			StorageVolumeSize:      "3Mib",
			StorageMaxChunksPause:  true,
			CPUBufferWorkers:       4,
			CPULimit:               "5Mib",
			CPURequest:             "6Mib",
			MemoryLimit:            "7Mib",
			MemoryRequest:          "8Mib",
			CreatedAt:              now.Add(-time.Hour),
		}, {
			ID:                     "resource_profile_id_2",
			Name:                   "name_2",
			StorageMaxChunksUp:     8,
			StorageSyncFull:        true,
			StorageBacklogMemLimit: "7Mib",
			StorageVolumeSize:      "6Mib",
			StorageMaxChunksPause:  true,
			CPUBufferWorkers:       5,
			CPULimit:               "4Mib",
			CPURequest:             "3Mib",
			MemoryLimit:            "2Mib",
			MemoryRequest:          "1Mib",
			CreatedAt:              now.Add(time.Minute * -30),
		}}
		got := &bytes.Buffer{}
		cmd := newCmdGetResourceProfiles(configWithMock(&ClientMock{
			ResourceProfilesFunc: func(ctx context.Context, aggregatorID string, params cloud.ResourceProfilesParams) ([]cloud.ResourceProfile, error) {
				return want, nil
			},
		}))
		cmd.SetArgs([]string{"--aggregator=" + zeroUUID4})
		cmd.SetOutput(got)

		err := cmd.Execute()
		wantEq(t, nil, err)
		wantEq(t, ""+
			"NAME   STORAGE-MAX-CHUNKS-UP STORAGE-SYNC-FULL STORAGE-BACKLOG-MEM-LIMIT STORAGE-VOLUME-SIZE STORAGE-MAX-CHUNKS-PAUSE CPU-BUFFER-WORKERS CPU-LIMIT CPU-REQUEST MEM-LIMIT MEM-REQUEST AGE\n"+
			"name_1 1                     true              2Mib                      3Mib                true                     4                  5Mib      6Mib        7Mib      8Mib        1 hour\n"+
			"name_2 8                     true              7Mib                      6Mib                true                     5                  4Mib      3Mib        2Mib      1Mib        30 minutes\n", got.String())

		t.Run("with_ids", func(t *testing.T) {
			got.Reset()
			cmd.SetArgs([]string{"--aggregator=" + zeroUUID4, "--show-ids"})

			err := cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, ""+
				"ID                    NAME   STORAGE-MAX-CHUNKS-UP STORAGE-SYNC-FULL STORAGE-BACKLOG-MEM-LIMIT STORAGE-VOLUME-SIZE STORAGE-MAX-CHUNKS-PAUSE CPU-BUFFER-WORKERS CPU-LIMIT CPU-REQUEST MEM-LIMIT MEM-REQUEST AGE\n"+
				"resource_profile_id_1 name_1 1                     true              2Mib                      3Mib                true                     4                  5Mib      6Mib        7Mib      8Mib        1 hour\n"+
				"resource_profile_id_2 name_2 8                     true              7Mib                      6Mib                true                     5                  4Mib      3Mib        2Mib      1Mib        30 minutes\n", got.String())
		})

		t.Run("json", func(t *testing.T) {
			want, err := json.Marshal(want)
			wantEq(t, nil, err)

			got.Reset()
			cmd.SetArgs([]string{"--aggregator=" + zeroUUID4, "--output-format=json"})

			err = cmd.Execute()
			wantEq(t, nil, err)
			wantEq(t, string(want)+"\n", got.String())
		})
	})
}

package main

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"errors"
// 	"testing"
// 	"time"

// 	"github.com/calyptia/api/types"
// )

// func Test_newCmdCreatePipelineFile(t *testing.T) {
// 	t.Run("required", func(t *testing.T) {
// 		cmd := newCmdCreatePipelineFile(configWithMock(nil))
// 		cmd.SilenceErrors = true
// 		cmd.SilenceUsage = true

// 		err := cmd.Execute()
// 		wantErrMsg(t, `required flag(s) "file", "pipeline" not set`, err)

// 		f := setupFile(t, "file.txt", nil)
// 		defer f.Close()

// 		cmd.SetArgs([]string{"--file", f.Name()})
// 		err = cmd.Execute()
// 		wantErrMsg(t, `required flag(s) "pipeline" not set`, err)
// 	})

// 	t.Run("find_error", func(t *testing.T) {
// 		want := errors.New("internal error")
// 		cmd := newCmdCreatePipelineFile(configWithMock(&ClientMock{
// 			ProjectPipelinesFunc: func(ctx context.Context, projectID string, params types.PipelinesParams) (types.Pipelines, error) {
// 				return types.Pipelines{}, want
// 			},
// 		}))
// 		cmd.SilenceErrors = true
// 		cmd.SilenceUsage = true

// 		f := setupFile(t, "file.txt", nil)
// 		defer f.Close()

// 		cmd.SetArgs([]string{"--pipeline", "foo", "--file", f.Name()})

// 		got := cmd.Execute()
// 		wantEq(t, want, got)
// 	})

// 	t.Run("create_error", func(t *testing.T) {
// 		want := errors.New("internal error")
// 		cmd := newCmdCreatePipelineFile(configWithMock(&ClientMock{
// 			ProjectPipelinesFunc: func(ctx context.Context, projectID string, params types.PipelinesParams) (types.Pipelines, error) {
// 				return types.Pipelines{
// 					Items: []types.Pipeline{{
// 						ID: "foo",
// 					}},
// 				}, nil
// 			},
// 			CreatePipelineFileFunc: func(ctx context.Context, pipelineID string, payload types.CreatePipelineFile) (types.CreatedPipelineFile, error) {
// 				return types.CreatedPipelineFile{}, want
// 			},
// 		}))
// 		cmd.SilenceErrors = true
// 		cmd.SilenceUsage = true

// 		f := setupFile(t, "file.txt", nil)
// 		defer f.Close()

// 		cmd.SetArgs([]string{"--pipeline", "foo", "--file", f.Name()})

// 		got := cmd.Execute()
// 		wantEq(t, want, got)
// 	})

// 	t.Run("ok", func(t *testing.T) {
// 		now := time.Now().Truncate(time.Second)
// 		want := types.CreatedPipelineFile{
// 			ID:        "want_file_id",
// 			CreatedAt: now.Add(-time.Minute),
// 		}
// 		wantPipelineID := "want_pipeline_id"
// 		got := &bytes.Buffer{}
// 		mock := &ClientMock{
// 			ProjectPipelinesFunc: func(ctx context.Context, projectID string, params types.PipelinesParams) (types.Pipelines, error) {
// 				return types.Pipelines{
// 					Items: []types.Pipeline{{
// 						ID: wantPipelineID,
// 					}},
// 				}, nil
// 			},
// 			CreatePipelineFileFunc: func(ctx context.Context, pipelineID string, payload types.CreatePipelineFile) (types.CreatedPipelineFile, error) {
// 				return want, nil
// 			},
// 		}
// 		cmd := newCmdCreatePipelineFile(configWithMock(mock))
// 		cmd.SetOut(got)

// 		wantContents := []byte("hello world")
// 		wantName := "want_file_name"
// 		f := setupFile(t, wantName+".txt", wantContents)
// 		defer f.Close()
// 		cmd.SetArgs([]string{"--pipeline", "foo", "--file", f.Name(), "--encrypt", "true"})
// 		err := cmd.Execute()
// 		wantEq(t, nil, err)
// 		wantEq(t, ""+
// 			"ID           AGE\n"+
// 			"want_file_id 1 minute\n", got.String())

// 		calls := mock.CreatePipelineFileCalls()
// 		wantEq(t, 1, len(calls))

// 		call := calls[0]
// 		wantEq(t, wantPipelineID, call.S)
// 		wantEq(t, wantName, call.CreatePipelineFile.Name)
// 		wantEq(t, wantContents, call.CreatePipelineFile.Contents)
// 		wantEq(t, true, call.CreatePipelineFile.Encrypted)

// 		t.Run("json", func(t *testing.T) {
// 			want, err := json.Marshal(want)
// 			wantEq(t, nil, err)

// 			got.Reset()
// 			cmd.SetArgs([]string{"--output-format=json"})

// 			err = cmd.Execute()
// 			wantEq(t, nil, err)
// 			wantEq(t, string(want)+"\n", got.String())
// 		})
// 	})
// }
